/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/9/1 17:18
* @Description: The file is for
***********************************************************************/

package errs

import "errors"

// proposal提案指Entry.Data

// ErrProposalDropped pot竞赛冠军的提案因为某些原因被丢弃了
var ErrProposalDropped = errors.New("pot proposal dropped")