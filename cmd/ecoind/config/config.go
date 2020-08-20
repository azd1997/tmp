package config

import (
	"encoding/json"
	"fmt"
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/p2p/peer"
	"io/ioutil"

	"github.com/azd1997/ecoin/common/utils"
)

// TODO: yaml配置文件。

// 总配置
type config struct {
	AC accountConfig `json:"account_config" yaml:"account_config"`
	CC chainConfig   `json:"chain_config" yaml:"chain_config"`
	PC p2pConfig     `json:"p2p_config" yaml:"p2p_config"`
	LC logConfig     `json:"log_config" yaml:"log_config"`
	RC rpcConfig     `json:"rpc_config" yaml:"rpc_config"`
	DC dbConfig      `json:"db_config" yaml:"db_config"`
}

// account配置
type accountConfig struct {
	Type int    `json:"type" yaml:"type"` // 账户存储于文件的形式：0表示明文；1表示密文。目前只有明文存储
	Path string `json:"path" yaml:"path"`
}

// chain配置
type chainConfig struct {
	ChainID       uint8  `json:"chain_id" yaml:"chain_id"` // 链标识
	BlockInterval int    `json:"block_interval" yaml:"block_interval"`
	Genesis       string `json:"genesis" yaml:"genesis"` // 创世区块信息
}

// p2p配置
type p2pConfig struct {
	IP       string `json:"ip" yaml:"ip"`
	Port     int    `json:"port" yaml:"port"`
	MaxPeers int    `json:"max_peers" yaml:"max_peers"`
	Seeds    []seed `json:"seeds" yaml:"seeds"`
}

// 日志配置
type logConfig struct {
	LogLevel int `json:"log_level" yaml:"log_level"`
	LogColor bool `json:"log_color" yaml:"log_color"`
}

// rpc配置
type rpcConfig struct {
	HTTPPort int `json:"http_port" yaml:"http_port"`
}

// db配置
type dbConfig struct {
	DbPath string `json:"db_path" yaml:"http_port"`
}

// 种子节点的字符串存储结构
type seed struct {
	Addr  string `json:"addr" yaml:"addr"`
	HexID string `json:"hex_id" yaml:"hex_id"`
}

func ParseConfig(cf string) (*config, error) {
	if len(cf) == 0 {
		return nil, fmt.Errorf("miss config file")
	}

	if err := utils.AccessCheck(cf); err != nil {
		return nil, err
	}

	jsonContent, err := ioutil.ReadFile(cf)
	if err != nil {
		return nil, fmt.Errorf("read config file failed:%v", err)
	}

	conf := &config{}
	if err := json.Unmarshal(jsonContent, &conf); err != nil {
		return nil, fmt.Errorf("config parse failed:%v", err)
	}

	if err := verifyConfig(conf); err != nil {
		return nil, err
	}

	return conf, nil
}

func verifyConfig(c *config) error {

	// TODO

	if c.RC.HTTPPort <= 0 || c.RC.HTTPPort > 65535 || c.RC.HTTPPort == c.PC.Port {
		return fmt.Errorf("invalid http port:%d", c.RC.HTTPPort)
	}

	return nil
}


// 解析种子列表。种子节点没有ID
func ParseSeeds(seeds []seed) []*peer.Peer {
	var result []*peer.Peer

	for _, seed := range seeds {
		// 解析IP、Port
		ip, port := utils.ParseIPPort(seed.Addr)
		if ip == nil {
			continue
		}
		// 解析ID
		idB, _ := encoding.FromHex(seed.HexID)
		seedId := crypto.ID(idB)
		// 新建seed peer节点
		p := peer.NewPeer(ip, port, seedId)
		result = append(result, p)
	}

	return result
}