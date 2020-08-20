/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/14 11:52
* @Description: The file is for
***********************************************************************/

package main

import (
	"fmt"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/azd1997/ecoin/store/db"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	rootCmd.AddCommand(dbCmd)
	dbCmd.PersistentFlags().StringP("path", "p", "./data/", "db path")
	dbCmd.PersistentFlags().StringP("output", "o", "", "output to where. default stdout")

	dbCmd.AddCommand(dbBlockCmd)
	dbBlockCmd.Flags().StringP("range", "r", "", "block height range")
	dbBlockCmd.Flags().StringP("hash", "x", "", "block hash")

	dbCmd.AddCommand(dbTxCmd)
	dbTxCmd.Flags().StringP("hash", "x", "", "tx hash")
	// 注意：不能使用短名"h"，被"help"的"h"占用了

}

var dbCmd = &cobra.Command{
	Use:"dbview",
	Short:"db browser",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

var dbBlockCmd = &cobra.Command{
	Use:   "block",
	Short: "db browser: view block via hash",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		// 1. 打开数据库
		dbpath := cmd.Flag("path").Value.String()
		if err = utils.AccessCheck(dbpath); err != nil {
			fmt.Println("db path access check failed: ", err)
			os.Exit(1)
		}
		if err = db.Init(dbpath); err != nil {
			fmt.Println("db init failed: ", err)
			os.Exit(1)
		}
		// 2. 确定输出路径
		out := cmd.Flag("output").Value.String()
		if out != "" {
			if err = utils.AccessCheck(out); err != nil {
				fmt.Println("output path access check failed: ", err)
				os.Exit(1)
			}

			output, err = os.OpenFile(out, os.O_RDWR|os.O_CREATE, 0755)
			if err != nil {
				fmt.Printf("open file %s failed:%v\n", out, err)
				os.Exit(1)
			}
			defer output.Close()
		} else {
			output = os.Stdout
		}
		// 3. 解析查询参数
		rangeArg, _ := cmd.Flags().GetString("range")
		hashArg, _ := cmd.Flags().GetString("hash")
		if (rangeArg == "" && hashArg == "") || (rangeArg != "" && hashArg != "") {
			fmt.Println("please specify block height range or block hash in hex format.")
			os.Exit(1)
		}

		// 4. 根据参数情况去查询数据库
		if rangeArg != "" {		// 高度范围查询
			err = rangeView(rangeArg)
		} else {	// 区块哈希
			err = blockView(hashArg)
		}
		if err != nil {
			fmt.Printf("error happens when view db: %v\n", err)
			os.Exit(1)
		}

	},
}

var dbTxCmd = &cobra.Command{
	Use:   "tx",
	Short: "db browser: view tx via hash",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		// 1. 打开数据库
		dbpath := cmd.Flag("path").Value.String()
		if err = utils.AccessCheck(dbpath); err != nil {
			fmt.Println("db path access check failed: ", err)
			os.Exit(1)
		}
		if err = db.Init(dbpath); err != nil {
			fmt.Println("db init failed: ", err)
			os.Exit(1)
		}
		// 2. 确定输出路径
		out := cmd.Flag("output").Value.String()
		if out != "" {
			if err = utils.AccessCheck(out); err != nil {
				fmt.Println("db path access check failed: ", err)
				os.Exit(1)
			}

			output, err = os.OpenFile(out, os.O_RDWR|os.O_CREATE, 0755)
			if err != nil {
				fmt.Printf("open file %s failed:%v\n", out, err)
				os.Exit(1)
			}
			defer output.Close()
		} else {
			output = os.Stdout
		}
		// 3. 解析查询参数
		txArg := cmd.Flag("hash").Value.String()

		// 4. 根据参数情况去查询数据库
		err = txView(txArg)
		if err != nil {
			fmt.Printf("error happens when view db: %v\n", err)
			os.Exit(1)
		}

	},
}



