/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/6 14:22
* @Description: The file is for
***********************************************************************/

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/azd1997/ecoin/account"
	"github.com/azd1997/ecoin/cmd/ecoind/config"
	log2 "github.com/azd1997/ecoin/common/log"
	"github.com/azd1997/ecoin/enode"
	"github.com/azd1997/ecoin/enode/bc"
	"github.com/azd1997/ecoin/p2p"
	"github.com/azd1997/ecoin/p2p/peer"
	"github.com/azd1997/ecoin/rpc"
	"github.com/azd1997/ecoin/store/db"
)

func main() {
	// 解析cobra命令
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}

	// 加载配置
	conf, err := config.ParseConfig(cfgFile)
	if err != nil {
		log.Fatal(err)
	}

	// 加载日志模块
	log2.SetLogLevel(conf.LC.LogLevel)
	log2.SetLogColor(conf.LC.LogColor)
	logger := log2.GetStdoutLog()

	// 加载账户
	var acc = &account.Account{}
	if err := acc.LoadFileWithJsonDecode(conf.AC.Path); err != nil {
		logger.Fatal("load account failed: %s", conf.AC.Path)
	}

	// p2p peer provider
	provider := peer.NewProvider(conf.PC.IP, conf.PC.Port, acc.UserId())
	seeds := config.ParseSeeds(conf.PC.Seeds)
	provider.AddSeeds(seeds)
	provider.Start()

	// p2p node
	nodeConfig := &p2p.Config{
		NodeIP:     conf.PC.IP,
		NodePort:   conf.PC.Port,
		Provider:   provider,
		MaxPeerNum: conf.PC.MaxPeers,
		Account:acc,
		ChainID:    conf.CC.ChainID,
	}
	node := p2p.NewNode(nodeConfig)
	node.Start()

	// 启动数据库模块
	if err = db.Init(conf.DC.DbPath); err != nil {
		logger.Fatal("init db failed:c%v\n", err)
	}
	logger.Info("database initialize successfully under the data path:c%s\n", conf.DC.DbPath)

	// enode核心模块启动
	enodeInstance := enode.NewEnode(&enode.Config{
		Node:         node,
		Account:acc,

		Config: &bc.Config{
			BlockInterval:       conf.CC.BlockInterval,
			Genesis:             conf.CC.Genesis,
		},
	})

	// 本地HTTP服务器启动（rpc）
	httpConfig := &rpc.Config{
		Port: conf.RC.HTTPPort,
		En:    enodeInstance,
	}
	httpServer := rpc.NewServer(httpConfig)
	httpServer.Start()

	// pprof性能监测
	if pprofPort != 0 {
		go func() {
			pprofAddress := fmt.Sprintf("localhost:%d", pprofPort)
			log.Println(http.ListenAndServe(pprofAddress, nil))
		}()
	}

	// 等待关闭信号
	sc := make(chan os.Signal)
	signal.Notify(sc, os.Interrupt)
	signal.Notify(sc, syscall.SIGTERM)
	select {
	case <-sc:
		logger.Infoln("Quiting......")
		httpServer.Stop()
		enodeInstance.Stop()
		node.Stop()
		db.Close()
		logger.Infoln("Bye!")
		return
	}
}



