/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/6 15:14
* @Description: The file is for
***********************************************************************/

package main

import (
	"github.com/azd1997/ecoin/cmd/ecli/config"
	"github.com/spf13/cobra"
	"log"
)

var (
	accountArg = ""
	blockArg = ""
	txArg = ""
)

func init() {
	rootCmd.AddCommand(queryCmd)

	queryCmd.AddCommand(queryAccountCmd)
	queryAccountCmd.Flags().StringVarP(&accountArg, "arg", "a", "", "query account arg")

	queryCmd.AddCommand(queryBlockCmd)
	queryBlockCmd.Flags().StringVarP(&blockArg, "arg", "a", "", "query block arg")

	queryCmd.AddCommand(queryTxCmd)
	queryTxCmd.Flags().StringVarP(&txArg, "arg", "a", "", "query tx arg")

}

var queryCmd = &cobra.Command{
	Use:"query",
	Short:"query info of account/block/tx",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO 不加参数则按默认配置启动
		cmd.Usage()
	},
}

var queryAccountCmd = &cobra.Command{
	Use:"account",
	Short:"query account via id in hex format",
	Run: func(cmd *cobra.Command, args []string) {

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

		err = client.queryAccount(accountArg)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

var queryBlockCmd = &cobra.Command{
	Use:"block",
	Short:"query block via id in hex format",
	Run: func(cmd *cobra.Command, args []string) {
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

		err = client.queryBlocks(blockArg)
		if err != nil {
			log.Fatalln(err)
		}
	},
}

var queryTxCmd = &cobra.Command{
	Use:"tx",
	Short:"query tx via id in hex format",
	Run: func(cmd *cobra.Command, args []string) {

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

		err = client.queryTx(txArg)
		if err != nil {
			log.Fatalln(err)
		}
	},
}
