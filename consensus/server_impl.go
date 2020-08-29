/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/8/26 17:03
* @Description: The file is for
***********************************************************************/

package consensus

import (
	"context"
	"github.com/azd1997/ecoin/protocol/core"
)

// Server实现
type serverImpl struct {

}

// CollectLoop 收集循环
func (s *serverImpl) CollectLoop(ctx context.Context) {
	txchan := ctx.Value("txchan").(chan *core.Tx)

	for {
		select {
		case tx := <- txchan:
			// 写入到UBTXP

		}
	}
}

func ()