package utils

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
)

/*********************************************************************************************************************
                                                    error相关
*********************************************************************************************************************/

// LogErr 记录错误
func LogErr(err error) {
	if err != nil {
		pc, filename, lineno, ok := runtime.Caller(1)
		if !ok {
			return
		}
		filename = filepath.Base(filename)
		callFunc := runtime.FuncForPC(pc).Name()
		callFunc = filepath.Base(callFunc)
		log.Printf(" [ERROR] (%s:%s:%d) %s\n", filename, callFunc, lineno, err)
	}
}

// LogErrAndExit 记录错误并退出进程
func LogErrAndExit(err error) {
	LogErr(err)
	os.Exit(1)
}
