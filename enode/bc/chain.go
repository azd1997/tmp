package bc

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/common/log"
	"github.com/azd1997/ecoin/protocol/core"
	"github.com/azd1997/ecoin/store/db"
	"github.com/azd1997/ego/epattern"
)


const (
	// 一次最多同步syncMaxBlocks个区块
	syncMaxBlocks uint64 = 128

	// alpha is the height difference used in manipulating the chain
	// 1. if the branch_a is 'alpha' higher than the branch_b, then removes branch_b from cache
	// 2. if the block forwardNum (fordward reference block number) is 1,
	//  and it is 'alpha' lower than the longest branch, then saves it to db and try to remove it from cache
	// 3. if the received block is 'alpha' lower than the branch, it won't be accepted
	alpha = 8

	// ReferenceBlocks 同一时间只会有最多ReferenceBlocks个区块存于缓存（内存）
	ReferenceBlocks = 20
)

var (
	// BlockInterval 期望的区块产生间隔
	BlockInterval           time.Duration
)


// 区块链
// 从bc.Block的角度看是本地的多叉树（便于访问和处理分叉）
// 从core.Block的角度是区块链网络的单链表
// 注意，区块的构建时间是ns级别的(因为s级粒度太粗了，不适合用来做定时器)。交易则是s级别的
type Chain struct {
	// 变动通知。 这是为了向外部通知“我最长链改变了”
	PassiveChangeNotify chan int64	// 本应是bool类型，但是为了告诉外界此时链上最新区块的时间，选择传递时间戳

	// 缓存中最老区块（但是各个分支不是从这个区块开始分叉的）
	oldestBlock   *block
	// 分支列表
	branches      []*branch
	// 最长分支
	longestBranch *branch
	// 最高高度
	lastHeight    uint64
	// 分支锁
	branchLock    sync.RWMutex
	// 待处理区块通道，有缓冲(16)
	pendingBlocks chan []*core.Block
	lm            *epattern.LoopMode
}

// NewChain 创建一条新Chain。只能调用一次
func NewChain() *Chain {
	return &Chain{
		// 变化通知。用于当前最长链切换成另一条的变动通知
		PassiveChangeNotify: make(chan int64, 1),
		// 同时最多有16个区块待处理
		pendingBlocks:       make(chan []*core.Block, 16),
		lm:                  epattern.NewLoop(1),
	}
}

type Config struct {
	BlockInterval       int
	Genesis             string
}

// Init 从数据库初始化Chain。只允许调用一次
func (c *Chain) Init(conf *Config) error {
	if !db.HasGenesis() {
		logger.Info("chain starts with empty database")
		if err := c.initGenesis(conf.Genesis); err != nil {
			logger.Warn("chain init failed:%v\n", err)
			return err
		}
		return nil
	}

	return c.initFromDB()
}

func (c *Chain) Start() {
	go c.loop()
	c.lm.StartWorking()
}

func (c *Chain) Stop() {
	c.lm.Stop()
}

// AddBlocks 添加若干个区块
func (c *Chain) AddBlocks(blocks []*core.Block, local bool) {
	if local {	// local表示是本地区块（自己构建的），直接添加就好
		c.addBlocks(blocks, local)
		return
	}
	c.pendingBlocks <- blocks	// 非本地区块（从别人那收来的区块）需要严格的检查
}

// LatestBlockHash 返回最长链（分支）的最新区块哈希。 这是缓存中的最高区块，而不是数据库中的最高区块
func (c *Chain) LatestBlockHash() crypto.Hash {
	c.branchLock.RLock()
	defer c.branchLock.RUnlock()
	return c.longestBranch.hash()
}

// GetSyncHash 获取用于同步的区块哈希和高度差
func (c *Chain) GetSyncHash(base crypto.Hash) (end crypto.Hash, heightDiff uint32, err error) {
	c.branchLock.RLock()
	defer c.branchLock.RUnlock()

	var hdiff uint32

	// 最长分支中查找该区块(base)
	if baseBlock := c.longestBranch.getBlock(base); baseBlock != nil {
		b := c.longestBranch.head
		if bytes.Equal(b.Hash, base) {
			return nil, 0, ErrAlreadyUpToDate{base}
		}

		endHash := b.Hash
		for {
			// 本次查找过程中发生了缓存写入数据库的操作
			if b == nil {
				return nil, 0, ErrFlushingCache{base}
			}
			// 找到目标base
			if bytes.Equal(b.Hash, base) {
				break
			}
			hdiff++
			b = b.prev
		}

		return endHash, hdiff, nil
	}

	// 最长分支（缓存中）找不到，去数据库找
	_, baseHeight, err := db.GetHeaderViaHash(base)
	if err != nil {
		return nil, 0, ErrHashNotFound{base}
	}

	// 获取数据库最高区块哈希
	_, dbLatestHeight, dbLatestHash, err := db.GetLatestHeader()
	if err != nil {
		return nil, 0, err
	}

	// 如果base区块过度落后，那么为了避免对方一次性请求过多区块数据，暂时只高度对方我比你高syncMaxBlocks
	if dbLatestHeight-baseHeight >= syncMaxBlocks {
		respHash, _ := db.GetHash(baseHeight + syncMaxBlocks)
		return respHash, uint32(syncMaxBlocks), nil
	}

	hdiff = uint32(dbLatestHeight - baseHeight)
	return dbLatestHash, hdiff, nil
}

// GetSyncBlocks 返回同步使用（就是处理SyncRespMsg之后A知道B比A长，就会向B请求这些区块，
// 而B会从自己的本地链上通过本函数获取这些区块）的区块blocks
// 查找区间是 (base, end]。 因为base是对方(A)的最高区块，人家已经有了
func (c *Chain) GetSyncBlocks(base crypto.Hash, end crypto.Hash, onlyHeader bool) ([]*core.Block, error) {
	c.branchLock.RLock()
	defer c.branchLock.RUnlock()

	var result []*core.Block

	// 在最长分支上搜索
	// 先看baseBlock和endBlock是否存在于最长分支，存在则在最长分支上将这些区块迭代出来
	baseBlock := c.longestBranch.getBlock(base)
	endBlock := c.longestBranch.getBlock(end)
	if baseBlock != nil && endBlock != nil && baseBlock.height < endBlock.height {
		iter := endBlock
		for {
			// base区块时退出迭代
			if iter.height == baseBlock.height {
				break
			}
			// 还没遇到base区块的时候就缓存中查到底了
			// 这说明在这次查询（迭代）的 过程中 该分支发生了“写入数据库”的变化，只能报个错出去，出去再处理
			if iter == nil {
				return nil, ErrFlushingCache{base}
			}

			// 按高度升序排列
			result = append([]*core.Block{iter.Block.ShallowCopy(onlyHeader)}, result...)
			iter = iter.prev
		}
		return result, nil
	}

	if baseBlock == nil {
		logger.Debug("cache not found base, search in db\n")
	} else if endBlock == nil {
		logger.Debug("cache not found end, search in db\n")
	} else {
		return nil, ErrInvalidBlockRange{fmt.Sprintf("block heigh error, base %d, end %d\n",
			baseBlock.height, endBlock.height)}
	}

	// 如果一开始，base/end就有其一（只能有一个）不在缓存，那么就得去数据库找
	sBaseBlock, baseHeight, _ := db.GetBlockViaHash(base)
	sEndBlock, endHeight, _ := db.GetBlockViaHash(end)
	// 如果数据库中base/end都找到了，那么在数据库取中间所有区块信息
	if sBaseBlock != nil && sEndBlock != nil && baseHeight < endHeight {
		for i := baseHeight + 1; i <= endHeight; i++ {
			sBlock, _, _ := db.GetBlockViaHeight(i)
			// 按高度升序
			result = append(result, sBlock.ShallowCopy(onlyHeader))
		}
		return result, nil
	}
	// 如果数据库找到了base，缓存中找到了end，那么说明发生了缓存写入数据库的操作
	// 这种情况下result也许有数据（说明满足了前面数据库寻找的条件），也许没有（数据库和缓存查找的条件都没满足）
	if sBaseBlock != nil && endBlock != nil {
		return result, ErrFlushingCache{base}
	}

	return nil, ErrHashNotFound{base}
}

// GetSyncBlockHash 返回每个分支上的末端区块（最新、最高）哈希
func (c *Chain) GetSyncBlockHash() []crypto.Hash {
	c.branchLock.Lock()
	defer c.branchLock.Unlock()

	var result []crypto.Hash
	for _, bc := range c.branches {
		result = append(result, bc.hash())
	}
	return result
}

// VerifyTx 在最长分支（最长链）上验证交易的有效性
func (c *Chain) VerifyTx(tx *core.Tx) error {
	c.branchLock.RLock()
	defer c.branchLock.RUnlock()

	if err := c.longestBranch.verifyTx(tx); err != nil {
		return err
	}

	return nil
}

// GetUnstoredBlocks 返回所有未存储的区块及其高度，并按高度降序排序
func (c *Chain) GetUnstoredBlocks() ([]*core.Block, []uint64) {
	c.branchLock.RLock()
	defer c.branchLock.RUnlock()

	var blocks []*core.Block
	var heights []uint64
	iter := c.longestBranch.head
	for {
		// 在缓存的分支中向前遍历到底了
		if iter == nil {
			break
		}
		// 当前区块已被存储，那更早的区块肯定也被存储了
		if iter.isStored() {
			break
		}

		blocks = append(blocks, iter.Block)
		heights = append(heights, iter.height)
		iter = iter.prev
	}

	return blocks, heights
}

// 获取最高区块的时间
func (c *Chain) GetLatestBlockTime() int64 {
	return c.longestBranch.head.Time
}

// 根据genesis区块字节数组的十六进制编码字符串，来初始化Chain
func (c *Chain) initGenesis(genesis string) error {
	var genesisB []byte
	var cb *core.Block
	var err error

	if genesisB, err = encoding.FromHex(genesis); err != nil {
		return err
	}

	cb = &core.Block{}
	if err = cb.Decode(bytes.NewReader(genesisB)); err != nil {
		return err
	}

	if err = db.PutGenesis(cb); err != nil {
		return err
	}

	// the genesis block height is 1
	c.initFirstBranch(newBlock(cb, 1, true))
	return nil
}

// 从数据库初始化一条已有的Chain
func (c *Chain) initFromDB() error {
	// 获取要缓存的区块高度范围
	var beginHeight uint64 = 1
	lastHeight, err := db.GetLatestHeight()
	if err != nil {
		logger.Warn("get latest height failed:%v\n", err)
		return err
	}
	if lastHeight > ReferenceBlocks {
		beginHeight = lastHeight - ReferenceBlocks // 只把最近的ReferenceBlocks个区块缓存
	}

	// 获取要缓存(存于Chain)的区块
	var blocks []*block
	for height := beginHeight; height <= lastHeight; height++ {
		cb, _, err := db.GetBlockViaHeight(height)
		if err != nil {
			return fmt.Errorf("height %d, broken db data for block", height)
		}

		blocks = append(blocks, newBlock(cb, height, true))
	}

	// 初始化第一条分支（认为ReferenceBlocks个之前已经固化了，不会再更改）
	bc := c.initFirstBranch(blocks[0])
	for i := 1; i < len(blocks); i++ {
		bc.add(blocks[i])
	}
	return nil
}

// 初始化第一条分支
func (c *Chain) initFirstBranch(b *block) *branch {
	bc := newBranch(b)
	c.oldestBlock = b
	c.branches = append(c.branches, bc)
	c.longestBranch = bc
	c.lastHeight = c.longestBranch.height()
	return bc
}

// 工作循环
func (c *Chain) loop() {
	c.lm.Add()
	defer c.lm.Done()

	// maintainTicker 保持的是区块链的最新版本(与网络中其他最新一致)
	maintainTicker := time.NewTicker(time.Duration(2) * BlockInterval)
	// statusReportTicker 通知报告状态
	statusReportTicker := time.NewTicker(BlockInterval / 2)

	for {
		select {
		case <-c.lm.D:
			return
		case <-maintainTicker.C:
			c.maintain()
		case blocks := <-c.pendingBlocks:
			c.addBlocks(blocks, false)
		case <-statusReportTicker.C:
			c.statusReport()
		}
	}
}

// maintain cleans up the chain and flush cache into db
// 清理chain，将已满的缓存写入数据库
func (c *Chain) maintain() {
	c.branchLock.Lock()
	defer c.branchLock.Unlock()

	// 将所有比最长分支短alpha(8)个以上区块的分支移除，剩下的保留(reservedBranches)
	var reservedBranches []*branch
	for _, bc := range c.branches {
		if c.longestBranch.height()-bc.height() > alpha {
			logger.Debug("remove branch %s\n", bc.String())
			bc.remove()
			continue
		}
		reservedBranches = append(reservedBranches, bc)
	}
	c.branches = reservedBranches

	// 从当前Chain缓存中的最老区块(oldestBlock)开始，如果都是单链（没有分叉）则写入数据库。
	// 这是因为前一步，会不断地将落后alpha的分支删除，所以最终都会形成单链，而下面做的就是把剩下的单链(最长链)
	// 的已确定部分写入到数据库
	iter := c.oldestBlock
	for {
		// 待写入数据库的每个区块必须保证其子区块数为1(符合单链定义)
		if iter.nextsNum() != 1 {
			break
		}

		// 如果从当前区块处并没有分叉，并且当前区块高度比最高小alpha，那么说明当前区块可以被写入到数据库固化
		if c.longestBranch.height()-iter.height > alpha {
			removingBlock := iter
			if !removingBlock.isStored() {
				if err := db.PutBlock(removingBlock.Block, removingBlock.height); err != nil {
					logger.Fatal("store block failed:%v\n", err)
				}
				removingBlock.stored = true
				logger.Debug("store block (height %d)\n", iter.height)
			}

			// iter游标移动到当前区块的下一个区块（注意当前区块只有一个子区块）
			removingBlock.nexts.Range(func(k, v interface{}) bool {
				vBlock := v.(*block)
				iter = vBlock
				return true
			})

			// 当区块存入数据库之后并不是立即从缓存删除
			// 而是要保证该区块比最高区块矮syncMaxBlocks
			// 区块同时存于缓存和数据库方便区块的同步（可以 更大概率 直接从缓存得到历史数据而不是去查数据库）
			if c.longestBranch.height()-iter.height > syncMaxBlocks {

				// 待删除区块removingBlock与原本子区块移除对彼此的引用
				removingBlock.removeNext(iter)
				iter.removePrev()

				// 从各个分支的缓存中移除removingBlock
				for _, bc := range c.branches {
					bc.removeFromCache(removingBlock)
				}

				// 更新oldest
				c.oldestBlock = iter
			}

		} else {	// 否则的话退出，暂时不把removingBlock从缓存删除
			break
		}
	}
}

// 添加区块
// 这里假定blocks是按照先后顺序添加的。也就是说blocks[0]应该是当前最高区块的下一个
func (c *Chain) addBlocks(blocks []*core.Block, local bool) {
	c.branchLock.Lock()
	defer c.branchLock.Unlock()

	if len(blocks) == 0 {
		logger.Warnln("add blocks failed: empty blocks")
		return
	}

	// 以blocks[0]为子区块（下一个区块），找到对应的分支bc
	// 如果没找到的话，说明blocks[0]可能已经出现过，那么需要从blocks[0]处创建新分支
	// 也有可能blocks[0]比当前最高区块都高超过1，那么创建新分支会失败，暂时不会将该blocks添加到chain中
	var err error
	var bc *branch
	lastHash := blocks[0].PrevHash
	bc = c.getBranch(lastHash)
	if bc == nil {
		if bc, err = c.createBranch(blocks[0]); err != nil {
			logger.Info("add blocks failed:%v\n", err)
			return
		}
	}

	// 将blocks添加到分支bc上
	for _, cb := range blocks {
		if err := bc.verifyBlock(cb); err != nil {
			logger.Warn("verify blocks failed:%v\n", err)
			return
		}
		bc.add(newBlock(cb, bc.height()+1, false))
	}

	// 如果不是本地区块的话，那么通知检查，是否要更新最长链等属性
	// TODO
	if !local {
		c.notifyCheck()
	}
}

// 根据分支最新（末）区块的哈希来获取该分支
func (c *Chain) getBranch(blochHash crypto.Hash) *branch {
	for _, b := range c.branches {
		if bytes.Equal(b.hash(), blochHash) {
			return b
		}
	}
	return nil
}

// 创建新分支
// 条件是：该区块
// 必须存在于已有的分支当中（或者说存在于缓存中）
// 不比最高区块落后alpha(8)（否则视为太老，不予创建分支）
// 当前区块的父区块不能是已有分支的末端区块（否则没必要创建分支，直接接上就好）
func (c *Chain) createBranch(newBlock *core.Block) (*branch, error) {
	var result *branch
	lastHash := newBlock.PrevHash

	for _, b := range c.branches {
		if matchBlock := b.getBlock(lastHash); matchBlock != nil {
			if b.height()-matchBlock.height > alpha {
				return nil, fmt.Errorf("the block is too old, branch height %d, block height %d",
					b.height(), matchBlock.height)
			}

			if matchBlock.isPrevOf(newBlock) {
				return nil, fmt.Errorf("duplicated new block")
			}

			logger.Info("branch fork happen at block %s height %d\n",
				encoding.ToHex(matchBlock.Hash), matchBlock.height)

			result = newBranch(matchBlock)
			c.branches = append(c.branches, result)
			return result, nil
		}
	}

	return nil, fmt.Errorf("not found branch for last hash %X", lastHash)
}

// 获取最长分支
func (c *Chain) getLongestBranch() *branch {
	var longestBranch *branch
	var height uint64
	for _, b := range c.branches {
		if b.height() > height {
			longestBranch = b
			height = b.height()
		} else if b.height() == height {
			// 随机选择一条分支
			if time.Now().Unix()%2 == 0 {
				longestBranch = b
			}
		}
	}
	return longestBranch
}

// 通知检查
func (c *Chain) notifyCheck() {
	longestBranch := c.getLongestBranch()
	// 如果当前最长分支的高度高于最高高度lastHeight，说明链增长了，要更新最长分支等
	if longestBranch.height() > c.lastHeight {
		c.longestBranch = longestBranch
		c.lastHeight = c.longestBranch.height()

		// 但是由于只有一条case语句，到这其实就等于c.PassiveChangeNotify <- true
		// 向c.PassiveChangeNotify写入信号。
		// 之所以有default，是为了避免当Notify通道已经写了一个数据（还没被读）此处阻塞
		select {
		case c.PassiveChangeNotify <- c.longestBranch.head.Time:		// 发送信号。将最长分支的最高区块构建时间传递出去
		default:
		}
	}
}

// 状态报告。 用于调试模式
// 每隔一定时间间隔打印当前Chain的状态
func (c *Chain) statusReport() {
	if log.GetLogLevel() < log.LogDebugLevel {
		return
	}

	c.branchLock.RLock()
	defer c.branchLock.RUnlock()

	branchNum := len(c.branches)
	text := "\n\toldest: %X with height %d \n\tlongest head:%X \n\tbranch number:%d, details:\n%s"

	var details string
	for i := 0; i < branchNum; i++ {
		details += c.branches[i].String() + "\n\n"
	}

	logger.Debug(text, c.oldestBlock.Hash[crypto.HASH_LENGTH-2:], c.oldestBlock.height,
		c.longestBranch.hash()[crypto.HASH_LENGTH-2:], branchNum, details)
}

