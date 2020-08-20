/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/6 14:26
* @Description: The file is for
***********************************************************************/

package main

import (
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	pprofPort int
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "./ecoind-config.json", "config file path")
	rootCmd.PersistentFlags().IntVarP(&pprofPort, "pprof", "p", 0, "pprof port")
}

var rootCmd = cobra.Command{
	Use:"ecoind",
	Short:"ecoind is a p2p node in ecoin network",
	Run: func(cmd *cobra.Command, args []string) {
		// TODO 不加参数则按默认配置启动
	},
}


