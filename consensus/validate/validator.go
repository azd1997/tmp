/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/9/8 9:39
* @Description: The file is for
***********************************************************************/

package validate

import "github.com/azd1997/ecoin/consensus/implement/pot/msgs"

// 字符串表示


////////////////// 验证器 ////////////////////
// 其具体实现需要用到区块链其他模块


// 验证器
// 验证器中有账号、区块链等指针，对收到的消息作检查
type Validator interface {
	// 检查完成后立即返回
	Validate(msg *msgs.Message) Result
}

type validatorImpl struct {
	// 账户等成员
}

func (v validatorImpl) Validate() Result {

}
