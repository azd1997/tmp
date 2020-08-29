/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/8/26 18:06
* @Description: The file is for
***********************************************************************/

package pot

import "errors"

// ErrProposalDropped pot竞赛冠军的提案（也就是区块）因为某些原因被丢弃了
var ErrProposalDropped = errors.New("pot proposal dropped")