/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/6 15:34
* @Description: The file is for
***********************************************************************/

package main

import (
	"fmt"
	"github.com/azd1997/ecoin/account"
	"github.com/azd1997/ecoin/cmd/ecli/config"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/spf13/cobra"
	"log"
	"path/filepath"
	"strings"
)

// 交易构造参数
var (
	typ uint8 = 0
	uncompleted uint8 = 0
	from string = ""
	to string = ""
	amount uint32 = 0
	payload string = ""
	prevTxId string = ""
	description string = ""
)

func init() {
	rootCmd.AddCommand(newCmd)

	newCmd.AddCommand(newTxCmd)

	newTxCmd.Flags().Uint8P("local", "l", 0, "local means that only generate tx but don't upload")
	newTxCmd.Flags().Uint8Var(&typ, "type", 1, "tx type")
	newTxCmd.Flags().Uint8Var(&uncompleted, "uncompleted", 0, "tx uncompleted?")
	newTxCmd.Flags().StringVar(&from, "from", "", "tx from")
	newTxCmd.Flags().StringVar(&to, "to", "", "tx to")
	newTxCmd.Flags().Uint32Var(&amount, "amount", 0, "tx amount")
	newTxCmd.Flags().StringVar(&payload, "payload", "", "tx payload")
	newTxCmd.Flags().StringVar(&prevTxId, "prevtx", "", "tx prevTxId")
	newTxCmd.Flags().StringVar(&description, "description", "", "tx description")

	newCmd.AddCommand(newAccountCmd)

	newAccountCmd.Flags().Uint8P("roleno", "r", 99, "Specify new account role")
	newAccountCmd.Flags().StringP("output", "o", "", "Specify new account output to where")
}

var newCmd = &cobra.Command{
	Use:"new",
	Short:"new account/tx",
	Run: func(cmd *cobra.Command, args []string) {
		//
		cmd.Usage()
	},
}

var newTxCmd = &cobra.Command{
	Use:"tx",
	Short:"new tx",
	Run: func(cmd *cobra.Command, args []string) {
		// Tx有很多参数，检查比较麻烦
		if len(args) == 0 {
			log.Fatalln("wrong args length")
		}

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

		// 执行方法

		coreTx, err := client.generateTx(typ, uncompleted, from, to, description, payload, prevTxId, amount)
		if err != nil {
			log.Fatalln(err)
		}

		// 输出coreTx的十六进制编码字符串
		// 这是为了方便将交易操作分成两段：第一段本地生成，第二段再上传
		fmt.Println("> 生成新交易：")
		fmt.Println("	  >	type: \t", coreTx.Type)
		fmt.Println("	  >	uncompleted: \t", coreTx.Uncompleted)
		fmt.Println("	  >	from: \t", coreTx.From)
		fmt.Println("	  >	to: \t", coreTx.To)
		fmt.Println("	  >	amount: \t", coreTx.Amount)
		fmt.Println("	  >	payload: \t", string(coreTx.Payload))
		fmt.Println("	  >	prevTxId: \t", encoding.ToHex(coreTx.PrevTxId))
		fmt.Println("	  >	description: \t", coreTx.Description)
		fmt.Println("> hex_block: \t", encoding.ToHex(coreTx.Encode()))

		// 检查local参数
		local, err := cmd.Flags().GetUint8("local")
		if err != nil {
			log.Fatalln(err)
		}

		if local == 0 {		// 0表否，那么在创建时就直接上传；1表示是，创建完了之后可以根据hex_block上传
			err = client.uploadTx(coreTx)
			if err != nil {
				log.Fatalln(err)
			}
		}
	},
}

var newAccountCmd = &cobra.Command{
	Use:"account",
	Short:"new account",
	Run: func(cmd *cobra.Command, args []string) {
		// 执行方法
		role, err := cmd.Flags().GetUint8("roleno")
		if err != nil {
			log.Fatalln(err)
		}
		out, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatalln(err)
		}

		// 检查output位置是否存在账户文件
		exists, err := utils.FileExists(out)
		if exists {
			log.Fatalln("account file already exists, please specify another path")
		}

		// 创建账户并存储
		newAcc, err := account.NewAccount(role)
		if err != nil {
			log.Fatalln(err)
		}
		jsonSuffix := strings.HasSuffix(out, "json")
		if jsonSuffix {
			err = newAcc.SaveFileWithJsonEncode(out)
		} else {
			err = newAcc.SaveFileWithGobEncode(out)
		}
		if err != nil {
			log.Fatalln(err)
		}
		absOut, _ := filepath.Abs(out)
		log.Printf("new account: %s, file path: %s\n", newAcc.UserId(), absOut)
	},
}
