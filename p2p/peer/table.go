package peer

import (
	"math/rand"
	"sort"
	"sync"
	"time"

	log "github.com/azd1997/ego/elog"

	"github.com/azd1997/ecoin/account"
)

// table 维护节点信息
type table interface {
	addPeers(p []*Peer, isSeed bool)
	getPeers(expect int, exclude map[account.UserId]bool) []*Peer
	exists(id account.UserId) bool

	getPeersToPing() []*Peer
	getPeersToGetNeighbours() []*Peer

	recvPing(p *Peer)
	recvPong(p *Peer)

	refresh()
}

const (
	coolingTime        = peerExpiredTime * 2
	coolingExpiredTime = 5 * time.Minute
)

type tableImp struct {
	self       account.UserId

	// seeds节点组为硬编码的种子节点，一般来讲认为是不会作出恶意行为的
	// 即便不诚实不可达也不会移入其他表
	//
	// 除了seeds以外，节点会在peers/expiredPeers/disHonestPeers
	// 中间移动。正常都在peers，一旦不可达则移入expiredPeers；
	// 一旦不诚实(通常不诚实的节点是可达的)将其移入dishonestPeers
	// 不诚实的节点即便不可达也不会移入expiredPeers
	seeds        map[account.UserId]*state
	peers        map[account.UserId]*state
	bannedPeers map[account.UserId]*state
	expiredPeers map[account.UserId]*state

	// 随机源
	r            *rand.Rand

	// 读写锁
	sync.RWMutex
}

func newTable(self account.UserId) table {
	return &tableImp{
		self:           self,
		seeds:          make(map[account.UserId]*state),
		peers:          make(map[account.UserId]*state),
		bannedPeers: make(map[account.UserId]*state),
		expiredPeers:   make(map[account.UserId]*state),
		r:              rand.New(rand.NewSource(time.Now().Unix())),
	}
}

// 批量添加节点，单个节点也使用该方法添加
func (t *tableImp) addPeers(p []*Peer, isSeed bool) {
	t.Lock()
	defer t.Unlock()

	for _, peer := range p {
		pst := newState(peer, isSeed)
		t.add(pst, isSeed)
	}
}

// 批量获取节点(无序)
func (t *tableImp) getPeers(expect int, exclude map[account.UserId]bool) []*Peer {
	var peers []*Peer

	t.Lock()
	for _, peer := range t.peers {
		if _, ok := exclude[peer.ID]; !ok && peer.isAvaible() {
			peers = append(peers, peer.Peer)
		}
	}
	t.Unlock()

	peerSize := len(peers)
	if peerSize <= expect {
		return peers
	}

	// 打乱
	for i := 0; i < peerSize; i++ {
		j := t.r.Intn(peerSize)
		peers[i], peers[j] = peers[j], peers[i]
	}

	return peers[:expect]
}


// 批量获取节点(按delay排序)
// 如果要获取全部可用的节点，直接给入一个超大的expect，或者设置expect = len(t.peers)
func (t *tableImp) getSortedPeers(expect int, exclude map[account.UserId]bool) []*Peer {
	var peers []state

	// 注意! 这样的值拷贝只是权宜之计
	// 一旦表过大，那么这查询的代价将会非常大

	t.RLock()
	for _, peer := range t.peers {
		if _, ok := exclude[peer.ID]; !ok && peer.isAvaible() {
			peers = append(peers, *peer)	// 注意是值拷贝
		}
	}
	t.RUnlock()

	// 对peers按照delay排序
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].dState.pingDelay < peers[i].dState.pingDelay
	})

	// 取前expect个
	n := len(peers)
	if n > expect {
		n = expect
	}

	res := make([]*Peer, n)
	for i:=0; i<n; i++ {
		res[i] = peers[i].Peer
	}

	return res
}

// exists 是否存在某节点
func (t *tableImp) exists(id account.UserId) bool {
	t.RLock()
	defer t.RUnlock()

	_, ok := t.peers[id]
	return ok
}

// 获取节点去ping
func (t *tableImp) getPeersToPing() []*Peer {
	t.Lock()
	defer t.Unlock()

	result := make([]*Peer, 0, len(t.peers) + len(t.seeds))
	for _, peer := range t.peers {
		if peer.isTimeToPing() {
			result = append(result, peer.Peer)
			peer.doPing()
		}
	}

	for _, seed := range t.seeds {
		result = append(result, seed.Peer)
	}

	return result
}

// 获取节点去询问邻居
func (t *tableImp) getPeersToGetNeighbours() []*Peer {
	t.Lock()
	defer t.Unlock()

	var result []*Peer
	for _, peer := range t.peers {
		if peer.isTimeToGetNeighbours() {
			result = append(result, peer.Peer)
			peer.updateGetNeighbourTime()
		}
	}
	return result
}

// 接收到ping的处理
func (t *tableImp) recvPing(p *Peer) {
	t.Lock()
	defer t.Unlock()

	if _, ok := t.peers[p.ID]; ok {
		return
	}

	// 移除可能的过期节点(不可达状态)，转移到peers下
	if peer, ok := t.expiredPeers[p.ID]; ok {
		peer.recoverFromExpired()
		t.peers[p.ID] = peer
		delete(t.expiredPeers, p.ID)
	}

	// 添加状态
	pst := newState(p, false)
	t.add(pst, false)
}

// 接收到pong的处理
func (t *tableImp) recvPong(p *Peer) {
	t.Lock()
	defer t.Unlock()

	// pong消息必然来自自身表中记录了的节点，否则不必理会

	if peer, ok := t.peers[p.ID]; ok {
		peer.updatePingDelayAndPingOKTime()
		return
	}

	if seed, ok := t.seeds[p.ID]; ok {
		seed.updatePingDelayAndPingOKTime()
		return
	}
}

// 刷新
// 检查节点是否不可达/不诚实/是否过期/是否解封
// 不诚实了要移入dishonest
// 过期了要移入expired
// 接收到ping了，要从expired中移回peers(由recvPing处理)
// 账号解封了，移入peers
func (t *tableImp) refresh() {
	t.Lock()
	defer t.Unlock()

	for _, peer := range t.peers {
		if !peer.isHonest() {
			log.Trace("p2p peer %v turn banned", peer.Peer)
			peer.turnBanned()
			t.bannedPeers[peer.ID] = peer
			delete(t.peers, peer.ID)
			continue
		}
		if !peer.isReachable() {
			log.Trace("p2p peer %v turn expired", peer.Peer)
			t.expiredPeers[peer.ID] = peer
			delete(t.peers, peer.ID)
			continue
		}
	}

	curr := time.Now()
	for _, peer := range t.bannedPeers {
		// 解封
		if curr.After(peer.cState.unbanTime) {
			peer.recoverFromBanned()
			t.peers[peer.ID] = peer
			delete(t.bannedPeers, peer.ID)
		}
	}
}

// add helper(should call with lock)
func (t *tableImp) add(pst *state, isSeed bool) {
	// 种子节点
	if isSeed {
		if _, ok := t.seeds[pst.ID]; !ok {
			t.seeds[pst.ID] = pst
			return
		} else {return}
	}

	// 非种子节点

	if _, ok := t.bannedPeers[pst.ID]; ok {
		return
	}
	if _, ok := t.expiredPeers[pst.ID]; ok {
		return
	}
	if pst.ID == t.self {
		return
	}

	if _, ok := t.peers[pst.ID]; !ok {
		log.Trace("add peer %v\n", pst)
		t.peers[pst.ID] = pst
	}
}

