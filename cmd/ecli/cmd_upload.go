/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/17 23:06
* @Description: The file is for
***********************************************************************/

package main

import (
	"bytes"
	"github.com/azd1997/ecoin/cmd/ecli/config"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/protocol/core"
	"github.com/spf13/cobra"
	"log"
)

func init() {
	rootCmd.AddCommand(uploadCmd)

	uploadCmd.AddCommand(uploadTxCmd)
	uploadTxCmd.Flags().StringP("hextx", "h", "", "hex encoded tx")
}

var uploadCmd = &cobra.Command{
	Use:"upload",
	Short:"upload encoded account/block/tx",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO 不加参数则按默认配置启动
		cmd.Usage()
	},
}

var uploadTxCmd = &cobra.Command{
	Use:"tx",
	Short:"upload tx in hex format",
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

		// 3. 读取命令行参数
		hexTxStr := cmd.Flag("hextx").Value.String()
		hexTx, err := encoding.FromHex(hexTxStr)
		if err != nil {
			log.Fatalln(err)
		}
		tx := &core.Tx{}
		if err = tx.Decode(bytes.NewReader(hexTx)); err != nil {
			log.Fatalln(err)
		}
		if err = client.uploadTx(tx); err != nil {
			log.Fatalln(err)
		}
	},
}
