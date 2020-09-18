/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/9/13 15:52
* @Description: The file is for
***********************************************************************/

package potpb


type MessageType uint64

const (
	MessageType_Proof MessageType = 0
	MessageType_Block MessageType = 1
)

// Message
type Message struct {
	// 消息类型
	MessageType MessageType

	// 消息的直接来源
	From string
	// 消息的直接接收
	To string
	// 消息链路
	MsgLine []string	// 简单的验证规则：if From != MsgLine[-1] 无效
	// 签名保护MsgLine 不被修改
	Sigs []string	// 所有签名采用十六进制字符串编码。 Sigs[0]保护所有信息，Sig[1]保护Sig[0]及MsgLine[1]，以此类推...

	// 消息中传递的条目列表。 MessageBlock
	Entries []*models.Entry

	// 传递的证明消息列表，通常为长度为1, MsgProof
	Proofs []*models.Proof
}