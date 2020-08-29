/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/8/28 19:54
* @Description: The file is for
***********************************************************************/

package pot

// PotLog 日志存储模块
type PotLog struct {

	// storage 存储所有稳定的entries，在上一次压缩快照之后
	// 应用于区块链中时，storage封装了一层区块链
	storage Storage

	//
	entries []Entry
}


//