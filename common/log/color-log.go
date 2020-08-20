package log

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)


// Prefix 日志前缀
type Prefix string

const (
	P_INFO Prefix = "[INFO]"
	P_TRAC Prefix = "[TRAC]"
	P_ERRO Prefix = "[ERRO]"
	P_WARN Prefix = "[WARN]"
	P_SUCC Prefix = "[SUCC]"
)

// Color 颜色
type Color uint8

func (c *Color) Color(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", c, s)
}

const (
	RED = Color(iota + 91)
	GREEN	// 92
	YELLOW	// 93
	BLUE	// 94
	MAGENTA		// 95
)

func Trace(format string, a ...interface{}) {
	prefix := yellow(string(P_TRAC))
	fmt.Println(formatLog(prefix), fmt.Sprintf(format, a...))
}

func Info(format string, a ...interface{}) {
	prefix := blue(string(P_INFO))
	fmt.Println(formatLog(prefix), fmt.Sprintf(format, a...))
}

func Success(format string, a ...interface{}) {
	prefix := green(string(P_SUCC))
	fmt.Println(formatLog(prefix), fmt.Sprintf(format, a...))
}

func Warn(format string, a ...interface{}) {
	prefix := magenta(string(P_WARN))
	fmt.Println(formatLog(prefix), fmt.Sprintf(format, a...))
	// TODO: 增加退出函数体操作
}

func Error(format string, a ...interface{}) {
	prefix := red(string(P_ERRO))
	fmt.Println(formatLog(prefix), fmt.Sprintf(format, a...))
	// TODO: 增加退出进程操作
}

func red(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", RED, s)
}
func green(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", GREEN, s)
}
func yellow(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", YELLOW, s)
}
func blue(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", BLUE, s)
}
func magenta(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", MAGENTA, s)
}

func formatLog(prefix string) string {
	return time.Now().Format("2006/01/02 15:04:05") + " " + prefix + " "
}


// ================================


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
		Error("(%s:%s:%d) %s\n", filename, callFunc, lineno, err)
	}
}

// LogErrAndExit 记录错误并退出进程
func LogErrAndExit(err error) {
	LogErr(err)
	os.Exit(1)
}
