/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/8/28 19:50
* @Description: The file is for
***********************************************************************/

package raft

// Progress 各节点的日志复制进度
type Progress struct {
	LastHash []byte	//
	LastIndex uint64
}

// Progress represents a follower’s progress in the view of the leader. Leader maintains
// progresses of all followers, and sends entries to the follower based on its progress.
type Progress struct {
	Match, Next uint64
}