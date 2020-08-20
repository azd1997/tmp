/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/6 14:22
* @Description: The file is for
***********************************************************************/

package main

import (
	"fmt"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:"version",
	Short:"Version of ecoind",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("ecoind v0.1")
	},
}
