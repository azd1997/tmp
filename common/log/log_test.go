/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/1 18:10
* @Description: The file is for
***********************************************************************/

package log

import (
	"github.com/pkg/errors"
	"testing"
)

func TestLogger(t *testing.T) {
	SetLogColor(true)
	logger := NewLogger("TEST")
	logger.Debugln("哈哈哈")
	logger.Info("%s", "呵呵呵")
	logger.Errorln(errors.New("错误"))

	SetLogColor(false)
	logger2 := NewLogger("TEST2")
	logger2.Debugln("哈哈哈")
	logger2.Info("%s", "呵呵呵")
	logger2.Errorln(errors.New("错误"))
}
