/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/8/26 17:34
* @Description: PoT共识中 阶段stage 的定义
***********************************************************************/

package pot

type Stage uint64

// 理解为一场比赛：发令枪响，每个人都报出自己的得分，所有人都收集完毕后，判断自己是不是冠军
const (
	StageCompete Stage = iota	// 开始pot竞争(广播自己的得分，即PoTMsg)
	StageCollectScores				// 收集所有人的得分
	StageJudge						// 判断自己是否是冠军
	StageBroadcastBlock		 		// 冠军广播区块
	StageWaitBlock					// 其他选手等待区块
)

var stmap = [...]string{
	"StageCompete",
	"StageCollectScores",
	"StageJudge",
	"StageBroadcastBlock",
	"StageWaitBlock",
}

func (st Stage) String() string {
	return stmap[st]
}