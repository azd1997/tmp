package peer

import (
	"math"
	"time"
)



// PeerState 由原先的EAddr概念演化而来
// 原先的EAddr问题在于所有状态都绑定子啊一张表中，使用时加锁过于频繁，粒度过粗
// 修改只能整个状态拷贝出来修改再赋值回去，
// PeerState 包括：
// PeerDelayState(PDS) 记录节点通信时延，用于节点的优先度排序考量因素之一
// PeerHonestState(PHS) 记录节点诚实状态，用于节点的优先度排序考量因素之一
// PeerCreditState(PCS) 记录节点信誉分状态，信誉分过低时会被设置为 !honest，直至冷却时间结束
// PeerBehaviourState(PBS) 记录节点行为，积极响应加信誉分，恶意响应或发信减信誉分
//
// 其他：
//
// PeerPOTState(PPS) 记录节点PotMsg状态，暂时搁置
//
// DEPRECATED: 这些状态会分成多张表处理，尽管提供了一个聚合体PeerState
//
// 所有状态名称前`Peer`被省略
//
// 对于peer包以外，不需要直接访问这些表，因此都设为私有
// 对于外部需要知道的信息，通过State的聚合体(上层结构)提供API返回
//
// TODO: 信誉机制暂时不使用


//////////////////////////// Const & Var /////////////////////////////

const (
	// 节点过期时间，隔这段时间仍没有ping成功则视为不可连接
	peerExpiredTime      = 35 * time.Second

	// 获取邻居节点周期
	getNeighbourInterval = 15 * time.Second

	// ping周期
	pingInterval         = 10 * time.Second

	// 初始的信誉分
	initCredit = 10

	// 刚创建节点状态信息时其pingDelay值默认为1s
	initPingDelay = 1 * time.Second

	// 初始的(默认的)节点封禁间隔 (1天)
	defaultBanDuration = 24 * time.Hour

	// 封禁间隔增长底数 2 ^ (i-1) i为第i次被封禁
	defaultBanBase = 2
)

var (
	initTimepoint = time.Unix(0, 0)
)


//////////////////////////// PeerState /////////////////////////////

// 节点总状态
type state struct {
	*Peer

	// 是否为硬编码的种子节点，是则永远不被移除
	// 在ecoin中，种子节点通常为项目发起者维护的一批节点
	isSeed               bool

	lastGetNeighbourTime time.Time

	dState *delayState
	cState *creditState
}

func newState(p *Peer, isSeed bool) *state {
	return &state{
		Peer:                 p,
		isSeed:               isSeed,
		lastGetNeighbourTime: initTimepoint,
		dState:&delayState{
			pingDelay:     initPingDelay,
			pingStartTime: initTimepoint,
			pingNum:       0,
			unreachable:   false,
		},
		cState:&creditState{
			badRecords:       nil,
			continuousBadNum: 0,
			totalBadNum:      0,
			credit:           initCredit,
			dishonest:           false,
			unbanTime:        initTimepoint,
			bannedNum:0,
		},
	}
}

// isTimeToPing 判断是否到时间ping
func (p *state) isTimeToPing() bool {
	// 上次ping的时刻到现在是否过了ping周期
	return time.Now().Sub(p.dState.pingStartTime) >= pingInterval
}

// isAvaible 检查节点是否 可达 && 诚实
func (p *state) isAvaible() bool {
	return p.isReachable()
}

// isReachable 检查是否可达
func (p *state) isReachable() bool {
	return time.Now().Sub(p.dState.lastPingOKTime) < peerExpiredTime
}

// isHonest 检查是否诚实
func (p *state) isHonest() bool {
	return true
}

// isTimeToGetNeighbours 判断是否到时间去查询邻居节点
func (p *state) isTimeToGetNeighbours() bool {
	return time.Now().Sub(p.lastGetNeighbourTime) >= getNeighbourInterval
}

// 暂不使用，由于信誉积分系统的设计，不能随意删除记录
func (p *state) isToRemove() bool {
	if !p.isAvaible() && p.dState.pingStartTime.After(initTimepoint) && !p.isSeed {
		return true
	}
	return false
}

// isToSetUnReachable 判断是否设置为不可达
func (p *state) isToSetUnReachable() {
	if time.Now().Sub(p.dState.lastPingOKTime) < peerExpiredTime {
		return
	}
	p.dState.unreachable = true
}

// 开始ping这个节点
func (p *state) doPing() {
	p.dState.pingStartTime = time.Now()

}

// 更新时延
func (p *state) updatePingDelayAndPingOKTime() {
	p.dState.pingDelay = time.Now().Sub(p.dState.pingStartTime)
	p.dState.lastPingOKTime = time.Now()
}

// 更新获取邻居节点的时间
func (p *state) updateGetNeighbourTime() {
	p.lastGetNeighbourTime = time.Now()
}

// 进入过期状态不需要方法，直接进行表的迁移就行

// 从过期状态恢复
func (p *state) recoverFromExpired() {
	p.dState.pingDelay = initPingDelay
	p.dState.lastPingOKTime = time.Now()
}

// 从封禁状态恢复
func (p *state) recoverFromBanned() {
	p.dState.pingDelay = initPingDelay
	p.dState.lastPingOKTime = time.Now()

	// 重新置为诚实
	p.cState.dishonest = false
	p.cState.badRecords = nil
	p.cState.continuousBadNum = 0
	p.cState.credit = initCredit
}

// 进入封禁状态
func (p *state) turnBanned() {
	p.cState.bannedNum += 1
	p.cState.unbanTime = time.Now().Add(defaultBanDuration *
		time.Duration(math.Pow(defaultBanBase, float64(p.cState.bannedNum - 1))))
}


//////////////////////////// DelayState /////////////////////////////


// 时延状态
// 时延状态由Ping-Pong统计，Ping不通则设置为不可达
type delayState struct {

	// 通信延迟。通信延迟只统计1s以内，1s超时，设为math.MaxInt64
	// 被标记了不可达的节点当主动与本节点连接时，更新其可达状态
	// 还没被Ping过，则设为1s
	pingDelay time.Duration

	// ping开始的时间(ns级)，用于pong回返后计算时延
	pingStartTime time.Time

	// 上次ping通的时间.
	// 如果 time.Now().Sub(lastPingOKTime) > peerExpiredInterval，
	// 那么标记为不可达
	lastPingOKTime time.Time

	// TODO: 是否需要
	pingNum int

	// 通信不可达？ 默认为false，表示可达
	unreachable bool
}

//////////////////////////// DelayState /////////////////////////////



// 信誉状态
type creditState struct {

	// 作恶记录链表 这个链表一直从头部插入。
	// （某种意义上是个栈）链头节点（也就是这个）就是最新的作恶记录
	badRecords  *BadRecord

	// 作恶数，避免遍历链表
	// 作恶记录链表的长度，节点作恶记录会持续记录，直至信誉分被扣光，节点被封禁。
	continuousBadNum int

	// 当节点发送某些特殊信息或者是赎金交易之后，恢复节点能力，
	// 此时，信誉分清零，重新开始记录作恶链表。但TotalBadNum会记录作恶总数
	totalBadNum int

	// 信誉分
	credit           int // 信誉分. 假定信誉分初始为10， 每出一个区块加1，TODO

	// Honest诚实与否。 当信誉分降为0后，dishonest=true
	dishonest    bool  // 诚实与否的结果

	// 第几次被封禁
	bannedNum int

	// 解封时间
	// 第一次封禁会封禁DEFAULT_BAN_DURATION,
	// 封禁时间到则会解禁。
	// 第i次封禁封禁时长会变为 DEFAULT_BAN_DURATION * 2^(i-1)
	unbanTime time.Time
}











