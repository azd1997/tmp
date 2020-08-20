/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/5 14:36
* @Description: The file is for
***********************************************************************/

package enode

import (
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/protocol/core"
	"github.com/azd1997/ego/epattern"
	"sync"
)

type proofPool struct {
	CleanC chan bool	// 接收外界的通知，清理表信息
	proofs map[crypto.ID]*core.PoTProof		// 证明表
	winner *core.PoTProof	// 胜者的证明
	lock sync.RWMutex		// 锁
	broadcast chan<- []*core.PoTProof	// 广播
	lm *epattern.LoopMode	// 循环模式
}

func NewProofPool() *proofPool {
	return &proofPool{
		proofs:make(map[crypto.ID]*core.PoTProof),
		lm:epattern.NewLoop(1),
	}
}

func (pp *proofPool) setBroadcastChan(c chan<- []*core.PoTProof) {
	pp.broadcast = c
}

func (pp *proofPool) start() {
	go func() {
		pp.lm.Add()
		defer pp.lm.Done()
		for {
			select {
			case <-pp.lm.D:
				return
			case <- pp.CleanC:
				pp.cleanUp()
			}
		}
	}()

	pp.lm.StartWorking()
}


func (pp proofPool) stop() {
	pp.lm.Stop()
}

////////////////////////////////////////////////

// 以后再考虑记录证明历史的问题
// 以及证明被篡改的安全性问题

func (pp *proofPool) addProof(proofs []*core.PoTProof, fromBroadcast bool) {
	// 全部插入表中
	for _, proof := range proofs {
		pp.insert(proof)
	}

	// 如果不是来自广播，需要广播出去
	if !fromBroadcast {
		pp.broadcast <- proofs
	}
}

func (pp *proofPool) insert(proof *core.PoTProof) {
	pp.lock.Lock()
	defer pp.lock.Unlock()

	// 插入表中
	if _, ok := pp.proofs[proof.From]; !ok {
		pp.proofs[proof.From] = proof
	}	// 暂时考虑以最早接收到的那一条证明为准，所以如果已经存在就跳过
	// 更新winner
	if pp.winner == nil {	// 还没设置winner则直接设置
		pp.winner = proof
	}
	// 否则需要比较
	if proof.GreaterThan(pp.winner) {
		pp.winner = proof
	}
}

// 清除掉所有的证明信息
func (pp *proofPool) cleanUp() {
	pp.lock.Lock()
	defer pp.lock.Unlock()

	// 清除表
	pp.proofs = make(map[crypto.ID]*core.PoTProof)
	// 清除winner
	pp.winner = nil
}

// 获取胜者
func (pp *proofPool) winnerProof() *core.PoTProof {
	pp.lock.RLock()
	defer pp.lock.RUnlock()

	return pp.winner
}
