package utils

import (
	"github.com/pkg/errors"
	"testing"
)

func TestErrors(t *testing.T) {
	err := errors.New("出错！")
	err = errors.Wrap(err, "山上")
	LogErr(err)
}
