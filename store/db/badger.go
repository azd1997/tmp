package db

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"
	"path/filepath"
	"time"

	"github.com/azd1997/ego/epattern"

	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/azd1997/ecoin/protocol/core"
	"github.com/azd1997/ecoin/protocol/storage"

	"github.com/dgraph-io/badger"
)

// NOTICE: 为了避免混淆，区块链交易简写为Tx, 数据库事务简写为Txn

var placeHolder = []byte("0")

type badgerDB struct {
	*badger.DB
	lm *epattern.LoopMode
}

func newBadger() *badgerDB {
	return &badgerDB{
		lm: epattern.NewLoop(1),
	}
}

// Init 初始化
func (b *badgerDB) Init(path string) error {
	var dbpath string
	var err error

	// 取绝对路径
	if dbpath, err = filepath.Abs(path); err != nil {
		return err
	}
	// 数据库路径存在与否
	if existed, _ := utils.DirExists(dbpath); !existed {
		return fmt.Errorf("dppath [%s] is not exists", dbpath)
	}
	// 数据库配置
	opts := badger.DefaultOptions(dbpath)
	opts = opts.WithLogger(nil)
	opts = opts.WithValueLogFileSize(512 << 20)
	opts = opts.WithMaxTableSize(32 << 20)
	// 开启数据库
	b.DB, err = badger.Open(opts)
	if err != nil {
		return b.wrapError(err)
	}
	// 启动数据库模块工作循环(GC循环)
	b.start()
	return nil
}

// Close 关闭工作循环，关闭数据库连接
func (b *badgerDB) Close() {
	b.stop()	// 关闭数据库GC循环
	b.DB.Close()	// 关闭数据库
}

// HasGenesis 检查是否含有创世区块
func (b *badgerDB) HasGenesis() bool {
	// rf read func
	rf := func(tx *badger.Txn) error {
		_, err := tx.Get(mGenesis)
		return err
	}

	err := b.View(rf)
	if err == nil {
		return true
	} else if err == badger.ErrKeyNotFound {
		return false
	} else {
		logger.Error("check genesis failed: %v\n", err)
		return false
	}
}

// PutGenesis 存储genesis创世区块
func (b *badgerDB) PutGenesis(block *core.Block) error {
	// wf write func
	wf := func(tx *badger.Txn) error {
		if err := tx.Set(mGenesis, placeHolder); err != nil {	// mGenesis只是用来标记是否含有创世区块；真正的创世区块应该通过高度去索引
			return err
		}

		if err := b.putBlockTxn(block, 1, tx); err != nil {
			return err
		}

		if err := b.updateLatestHeightTxn(1, tx); err != nil {
			return err
		}

		return nil
	}

	return b.update(wf)
}

// PutBlock 存储区块数据。
// 区块高度递增且不允许修改其他原本存在的区块
func (b *badgerDB) PutBlock(block *core.Block, height uint64) error {
	// 检查新增区块高度是否有效
	latestHeight, err := b.GetLatestHeight()
	if err != nil {
		return err
	}
	expectHeight := latestHeight + 1
	if height != expectHeight {
		return ErrInvalidHeight{height, expectHeight}
	}
	// 存入区块并更新最高高度信息
	wf := func(txn *badger.Txn) error {
		if err := b.putBlockTxn(block, height, txn); err != nil {
			return err
		}

		if err := b.updateLatestHeightTxn(height, txn); err != nil {
			return err
		}

		return nil
	}

	return b.update(wf)
}

// GetHash 根据区块高度获取区块哈希
func (b *badgerDB) GetHash(height uint64) (crypto.Hash, error) {
	var result []byte
	hashKey := getHashKey(height)

	rf := func(txn *badger.Txn) error {
		item, err := txn.Get(hashKey)
		if err != nil {
			return err
		}
		// 值复制。（避免直接使用值）
		result, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}

		return nil
	}

	return result, b.view(rf)
}

// GetHeaderViaHeight 通过区块高度获取区块头数据，同时返回区块哈希
func (b *badgerDB) GetHeaderViaHeight(height uint64) (*core.BlockHeader, crypto.Hash, error) {
	hash, err := b.GetHash(height)
	if err != nil {
		return nil, nil, err
	}

	header, err := b.getHeader(height, hash)
	if err != nil {
		return nil, nil, err
	}

	return header.BlockHeader, hash, nil
}

// GetHeaderViaHash 通过区块哈希获取区块头数据，同时返回区块高度
func (b *badgerDB) GetHeaderViaHash(h crypto.Hash) (*core.BlockHeader, uint64, error) {
	height, err := b.getHeaderHeight(h)
	if err != nil {
		return nil, 0, err
	}

	header, err := b.getHeader(height, h)
	if err != nil {
		return nil, 0, err
	}

	return header.BlockHeader, height, nil
}

// GetBlockViaHeight 通过区块高度获取区块内容，同时返回区块哈希
func (b *badgerDB) GetBlockViaHeight(height uint64) (*core.Block, crypto.Hash, error) {
	hash, err := b.GetHash(height)
	if err != nil {
		return nil, nil, err
	}

	result, err := b.getCoreBlock(height, hash)
	if err != nil {
		return nil, nil, err
	}

	return result, hash, err
}

// GetBlockViaHash 通过区块哈希获取区块内容，同时返回区块高度
func (b *badgerDB) GetBlockViaHash(h crypto.Hash) (*core.Block, uint64, error) {
	height, err := b.getHeaderHeight(h)
	if err != nil {
		return nil, 0, err
	}

	result, err := b.getCoreBlock(height, h)
	if err != nil {
		return nil, 0, err
	}

	return result, height, nil
}

// GetTxViaHash 根据交易哈希获取交易内容，同时返回交易所在区块的高度
func (b *badgerDB) GetTxViaHash(h crypto.Hash) (*core.Tx, uint64, error) {
	height, err := b.getTxHeight(h)
	if err != nil {
		return nil, 0, err
	}

	tx, err := b.getTx(height, h)
	if err != nil {
		return nil, 0, err
	}

	return tx.Tx, height, nil
}

// GetTxFromHashesViaID 根据ID获取其作为发送方的所有交易的哈希，同时返回交易所在区块的高度
func (b *badgerDB) GetTxFromHashesViaID(id crypto.ID) ([]crypto.Hash, []uint64, error) {
	var txHashes [][]byte
	var heights []uint64

	rf := func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := getAccountTxFromKeyPrefix(id)
		prefixLen := len(prefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {	// 按前缀迭代Key
			item := it.Item()

			k := item.Key()	// key :  prefix + hash		// prefix : id + f(suffix)
			hash := make([]byte, len(k)-prefixLen)
			copy(hash, k[prefixLen:])
			txHashes = append(txHashes, hash)

			err := item.Value(func(v []byte) error {	// value : height
				heights = append(heights, byteh(v))
				return nil
			})

			if err != nil {
				return err
			}
		}
		return nil
	}

	if err := b.View(rf); err != nil {
		return nil, nil, b.wrapError(err)
	}

	return txHashes, heights, nil
}

// GetTxToHashesViaID 根据ID获取其作为发送方的所有交易的哈希，同时返回交易所在区块的高度
func (b *badgerDB) GetTxToHashesViaID(id crypto.ID) ([]crypto.Hash, []uint64, error) {
	var txHashes [][]byte
	var heights []uint64

	rf := func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		// 得到key前缀
		prefix := getAccountTxToKeyPrefix(id)
		prefixLen := len(prefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {		// 按前缀迭代Key
			item := it.Item()

			k := item.Key()		// key :  prefix + hash		// prefix : id + t(suffix)
			hash := make([]byte, len(k)-prefixLen)
			copy(hash, k[prefixLen:])
			txHashes = append(txHashes, hash)

			err := item.Value(func(v []byte) error {	// value : height
				heights = append(heights, byteh(v))
				return nil
			})

			if err != nil {
				return err
			}
		}
		return nil
	}

	if err := b.View(rf); err != nil {
		return nil, nil, b.wrapError(err)
	}

	return txHashes, heights, nil
}

// HasTx 查看数据库中是否存在某个交易
func (b *badgerDB) HasTx(h crypto.Hash) bool {
	rf := func(tx *badger.Txn) error {
		key := getTxHeightKey(h)
		_, err := tx.Get(key)
		return err
	}

	err := b.View(rf)
	if err == nil {
		return true
	} else if err == badger.ErrKeyNotFound {
		return false
	} else {
		logger.Warn("check tx failed:%v\n", err)
		return true
	}
}

// GetBalanceViaKey 通过ID来获取其账户下的余额
func (b *badgerDB) GetBalanceViaID(id crypto.ID) (uint64, error) {
	var result uint64

	rf := func(txn *badger.Txn) error {
		balanceKey := getBalanceKey(id)
		item, err := txn.Get(balanceKey)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			result = byteh(val)
			return nil
		})
	}

	err := b.View(rf)
	if err == badger.ErrKeyNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, b.wrapError(err)
	}
	return result, nil
}

// GetLatestHeight 获取最新的区块高度
func (b *badgerDB) GetLatestHeight() (uint64, error) {
	var result uint64

	rf := func(txn *badger.Txn) error {
		item, err := txn.Get(mLatestHeight)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			result = byteh(val)
			return nil
		})
	}

	return result, b.view(rf)
}

// GetLatestHeader 获取最新的区块头core.BlockHeader，及其高度与哈希
func (b *badgerDB) GetLatestHeader() (*core.BlockHeader, uint64, crypto.Hash, error) {
	lastHeight, err := b.GetLatestHeight()
	if err != nil {
		return nil, 0, nil, err
	}

	header, hash, err := b.GetHeaderViaHeight(lastHeight)
	if header == nil {
		return nil, 0, nil, err
	}

	return header, lastHeight, hash, nil
}


/////////////////////////////////////////////////////////

// 获取区块头所在的高度信息
func (b *badgerDB) getHeaderHeight(hash crypto.Hash) (uint64, error) {
	var result uint64
	headerHeightKey := getHeaderHeightKey(hash)

	rf := func(txn *badger.Txn) error {
		item, err := txn.Get(headerHeightKey)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			result = byteh(val)
			return nil
		})
	}

	return result, b.view(rf)
}

// 获取交易所在的区块高度
func (b *badgerDB) getTxHeight(hash crypto.Hash) (uint64, error) {
	var result uint64
	txHeightKey := getTxHeightKey(hash)

	rf := func(txn *badger.Txn) error {
		item, err := txn.Get(txHeightKey)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			result = byteh(val)
			return nil
		})
	}

	return result, b.view(rf)
}

// 获取storage.BlockHeader(包含core.BlockHeader以及高度信息)
func (b *badgerDB) getHeader(height uint64, hash crypto.Hash) (*storage.BlockHeader, error) {
	var result *storage.BlockHeader
	headerKey := getHeaderKey(height, hash)

	rf := func(txn *badger.Txn) error {
		item, err := txn.Get(headerKey)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			result = &storage.BlockHeader{}
			return result.Decode(bytes.NewReader(val))
		})
	}

	return result, b.view(rf)
}

// 获取storage.Block（只包含交易哈希的列表）
func (b *badgerDB) getBlock(height uint64, hash crypto.Hash) (*storage.Block, error) {
	var result *storage.Block
	blockKey := getBlockKey(height, hash)

	rf := func(txn *badger.Txn) error {
		item, err := txn.Get(blockKey)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			result = &storage.Block{}
			return result.Decode(bytes.NewReader(val))
		})
	}

	return result, b.view(rf)
}

// 获取storage.Tx(和core.Tx内容上是一样的)
func (b *badgerDB) getTx(height uint64, hash crypto.Hash) (*storage.Tx, error) {
	var result *storage.Tx
	txKey := getTxKey(height, hash)

	rf := func(txn *badger.Txn) error {
		item, err := txn.Get(txKey)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			result = &storage.Tx{}
			return result.Decode(bytes.NewReader(val))
		})
	}

	return result, b.view(rf)
}

// 根据区块高度和哈希获取core.Block
func (b *badgerDB) getCoreBlock(height uint64, hash crypto.Hash) (*core.Block, error) {
	// 获取区块头
	header, err := b.getHeader(height, hash)
	if err != nil {
		return nil, err
	}

	// 获取区块体内所有交易
	var txs []*core.Tx
	if !header.BlockHeader.IsEmptyMerkleRoot() {
		storageBlock, err := b.getBlock(height, hash)
		if err != nil {
			return nil, err
		}

		for _, txHash := range storageBlock.TxHashes {
			tx, err := b.getTx(height, txHash)
			if err != nil {
				return nil, err
			}
			txs = append(txs, tx.Tx)
		}
	}

	return &core.Block{
		BlockHeader: header.BlockHeader,
		Txs:        txs,
	}, nil
}

////////////////////////////////////// 事务 /////////////////////////////////////////

// 用来插入新区块的事务
func (b *badgerDB) putBlockTxn(block *core.Block, height uint64, txn *badger.Txn) error {
	hash := block.Hash
	header := storage.NewBlockHeader(block.BlockHeader, height)

	// 存储区块内的交易列表
	if !block.IsEmptyMerkleRoot() {
		if err := b.putTxTxn(hash, block.Txs, height, txn); err != nil {
			return err
		}
	} else {
		header.SetEmptyMerkleRoot()
	}

	// 存储高度+哈希到区块头的映射、高度到区块头哈希的映射、哈希到高度的映射
	storageData := header.Encode()

	if err := txn.Set(getHeaderKey(height, hash), storageData); err != nil {
		return err
	}
	if err := txn.Set(getHashKey(height), hash); err != nil {
		return err
	}
	if err := txn.Set(getHeaderHeightKey(hash), hbyte(height)); err != nil {
		return err
	}

	// TODO: 更新余额，为区块构建者提供奖励，这里先随便设置了一个1
	// TODO: 这里被废除，区块构建者的奖励由coinbase交易提供
	//if err := b.updateBalanceTxn(block.CreateBy, 1, txn); err != nil {
	//	return err
	//}

	return nil
}

// 用来批量插入新交易的事务
func (b *badgerDB) putTxTxn(hash crypto.Hash, txs []*core.Tx, height uint64, txn *badger.Txn) error {
	var txHashes [][]byte
	for _, tx := range txs {
		storageData := tx.Encode()

		// 存入交易本身
		if err := txn.Set(getTxKey(height, tx.Id), storageData); err != nil {
			return err
		}
		// 存入交易所在区块高度
		if err := txn.Set(getTxHeightKey(tx.Id), hbyte(height)); err != nil {
			return err
		}
		// 更新账户与交易的关联，更新发起方/接收方的余额
		if tx.From != crypto.ZeroID {
			if err := b.updateAccountTxFromTxn(tx, height, txn); err != nil {
				return err
			}
			if err := b.updateBalanceTxn(tx.From, -int64(tx.Amount), txn); err != nil {
				return err
			}
		}
		if tx.To != crypto.ZeroID {
			if err := b.updateAccountTxToTxn(tx, height, txn); err != nil {
				return err
			}
			if err := b.updateBalanceTxn(tx.To, int64(tx.Amount), txn); err != nil {
				return err
			}
		}


		txHashes = append(txHashes, tx.Hash())
	}

	// 存储“区块体”数据：这个区块体只包含交易列表的哈希
	block := storage.NewBlock(txHashes)
	storageData := block.Encode()

	if err := txn.Set(getBlockKey(height, hash), storageData); err != nil {
		return err
	}

	return nil
}

// 用来更新某账户作为发送方的交易的事务
func (b *badgerDB) updateAccountTxFromTxn(tx *core.Tx, height uint64, txn *badger.Txn) error {
	accountTxKey := append(getAccountTxFromKeyPrefix(tx.From), tx.Hash()...)
	heightValue := hbyte(height)
	return txn.Set(accountTxKey, heightValue)
}

// 用来更新某账户作为接收方的交易的事务
func (b *badgerDB) updateAccountTxToTxn(tx *core.Tx, height uint64, txn *badger.Txn) error {
	accountTxKey := append(getAccountTxToKeyPrefix(tx.From), tx.Hash()...)
	heightValue := hbyte(height)
	return txn.Set(accountTxKey, heightValue)
}

// 用来更新最高高度的事务
func (b *badgerDB) updateLatestHeightTxn(height uint64, txn *badger.Txn) error {
	if err := txn.Set(mLatestHeight, hbyte(height)); err != nil {
		return err
	}
	return nil
}

// inc为余额增量，根据交易获得增量数额。如果inc是负数，传入inc之前，要检查余额是否足够
func (b *badgerDB) updateBalanceTxn(id crypto.ID, inc int64, txn *badger.Txn) error {
	balanceKey := getBalanceKey(id)

	item, err := txn.Get(balanceKey)
	if err != nil && err != badger.ErrKeyNotFound {
		return err
	}

	origin := int64(0)
	if err != badger.ErrKeyNotFound {
		item.Value(func(val []byte) error {
			origin = int64(byteh(val))
			return nil
		})
	}

	if inc < 0 && -inc > origin {
		return errors.New("not sufficient balance")
	}
	origin += inc
	return txn.Set(balanceKey, hbyte(uint64(origin)))
}

//////////////////////////////////////////////////////////////////////////////

// BadgerDB增查API包装

func (b *badgerDB) view(fn func(txn *badger.Txn) error) error {
	return b.wrapError(b.View(fn))
}

func (b *badgerDB) update(fn func(txn *badger.Txn) error) error {
	return b.wrapError(b.Update(fn))
}

// wrap the error directly get from badger
func (b *badgerDB) wrapError(err error) error {
	if err == nil {
		return nil
	}

	if err == badger.ErrKeyNotFound {
		return ErrNotFound
	}

	logger.Warn("badger got unexpect err: %v\n", err)
	return ErrInternal
}

///////////////////////////////////////////////////////////////////////////////

// LoopMode

func (b *badgerDB) start() {
	go b.gcLoop()
	b.lm.StartWorking()
}

func (b *badgerDB) stop() {
	b.lm.Stop()
}

func (b *badgerDB) gcLoop() {
	b.lm.Add()
	defer b.lm.Done()

	ticker := time.NewTicker(10 * time.Minute)

	for {
		select {
		case <-b.lm.D:
			return
		case <-ticker.C:	// 每10分钟进行一次value log的垃圾回收
			b.RunValueLogGC(0.5)
		}
	}
}
