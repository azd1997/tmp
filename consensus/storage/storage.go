/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/8/26 18:18
* @Description: The file is for
***********************************************************************/

package storage

type Storage interface {
	// 从存储引擎中加载两种状态
	// HardState指区块链（也就是日志记录的最新区块、高度等信息）
	// ConfState指共识节点集群信息
	InitialState() (HardState, ConfState, error)
	// 范围读取entry，entry是区块的包装
	Entries(lo, hi uint64) ([]Entry, error)
	// LastIndex 最后一个区块的高度，区块高度从1开始
	LastIndex() (uint64, error)
}

