/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/6 15:01
* @Description: The file is for
***********************************************************************/

package main

import (
	"fmt"
	"github.com/azd1997/ecoin/account"
	"github.com/azd1997/ecoin/cmd/ecli/config"
	"github.com/azd1997/ecoin/common/log"
)

func init() {
	log.SetLogLevel(log.LogDebugLevel)
	log.SetLogColor(false)
}

func main() {
	rootCmd.Execute()
}

func initHTTPClient(conf *config.Config) (*httpClient, error) {
	var err error

	// TODO: account的两种存储形式：加密或不加密
	acc, err := account.LoadOrCreateAccount(conf.AccountPath)
	if err != nil {
		return nil, fmt.Errorf("restore account failed:%v", err)
	}

	return newHTTPClient(conf.ServerIP,
		conf.ServerPort, conf.Scheme, acc), nil
}

