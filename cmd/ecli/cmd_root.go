/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/6 15:05
* @Description: The file is for
***********************************************************************/

package main

import (
	"github.com/azd1997/ecoin/cmd/ecli/config"
	"github.com/spf13/cobra"
	"log"
)

var (
	cfgFile string
	client *httpClient
)

func init() {

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "./ecli-config.json", "config file path")

}

var rootCmd = cobra.Command{
	Use:"ecli",
	Short:"ecli is a commandline client of ecoind",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO 不加参数则按默认配置启动

		// 1. 读取参数cfgFile
		conf, err := config.ParseConfig(cfgFile)
		if err != nil {
			log.Fatalln(err)
		}
		// 2. 启动HTTP客户端
		client, err = initHTTPClient(conf)
		if err != nil {
			log.Fatalln(err)
		}
		//fmt.Println(client.scheme, client.serverIP, client.serverPort)
	},
}

