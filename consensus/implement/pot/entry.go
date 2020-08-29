/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/8/26 18:54
* @Description: Entry(条目)，共识集群中日志的基本单位
***********************************************************************/

package pot

// Entry Entry(条目)，共识集群中日志的基本单位
type Entry struct {
	EntryType EntryType
	// 存储区块数据
	Data []byte
}

