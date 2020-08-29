/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/8/26 17:08
* @Description: The file is for
***********************************************************************/

package consensus

import (
	"context"
	"time"
)

// context value键值对约定
//
// author: eiger
// result:
// txchan: chan *Tx


// BackgroundContext 添加了author-eiger这对键值对，用以校验
var BackgroundContext = context.WithValue(context.Background(), "author", "eiger")

func NewContext(key, val interface{}) context.Context {
	return context.WithValue()
}

// consensusContext context上下文
type consensusContext struct {

}

func (consensusContext) Deadline() (deadline time.Time, ok bool) {
	panic("implement me")
}

func (consensusContext) Done() <-chan struct{} {
	panic("implement me")
}

func (consensusContext) Err() error {
	panic("implement me")
}

func (consensusContext) Value(key interface{}) interface{} {
	panic("implement me")
}
