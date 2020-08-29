/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/8/26 17:28
* @Description: The file is for
***********************************************************************/

// PoT基于以下假设：
// 每个节点持有其他所有节点的地址信息。（实际情况是，可能缺少部分）

package pot

// Pot 代表一个Pot共识节点
type Pot struct {
	// 当前处于哪个阶段
	Stage Stage

	// PoT节点标识
	id string

	// PotLog 日志存储
	PotLog *PotLog

	// 日志复制进度
	Prs map[string]*Progress

	// scores 竞赛分数
	scores map[string]*PotScore

	// msgs 待发送的消息，本地消息以及网络消息都写到该队列
	msgs []Message

	// Winner 当前轮pot竞赛的胜者，在下一轮竞赛开启时重置
	Winner string


}

func newPot(c Config) *Pot {

}