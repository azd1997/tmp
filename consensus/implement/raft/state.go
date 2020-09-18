/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/8/26 18:47
* @Description: 存储（也就是区块链）的状态定义，包括链状态和集群配置状态
***********************************************************************/

package raft

type HardState struct {

}

type ConfState struct {
	Nodes []string
}





// StateType represents the role of a node in a cluster.
type StateType uint64

const (
	StateFollower StateType = iota
	StateCandidate
	StateLeader
)

var stmap = [...]string{
	"StateFollower",
	"StateCandidate",
	"StateLeader",
}

func (st StateType) String() string {
	return stmap[uint64(st)]
}