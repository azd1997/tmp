/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/13 7:54
* @Description: The file is for
***********************************************************************/

package config

import (
	"encoding/json"
	"fmt"
	"github.com/azd1997/ecoin/common/utils"
	"io/ioutil"
)

type Config struct {
	ServerIP     string `json:"server_ip"`
	ServerPort   int    `json:"server_port"`
	Scheme       string `json:"scheme"`
	IgnoreHidden int    `json:"ignore_hidden"`
	AccountType  int    `json:"account_type"`
	AccountPath  string `json:"account_path"`
}

func ParseConfig(cf string) (*Config, error) {
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

	conf := &Config{}
	if err := json.Unmarshal(jsonContent, &conf); err != nil {
		return nil, fmt.Errorf("config parse failed:%v", err)
	}

	if err := verifyConfig(conf); err != nil {
		return nil, err
	}

	return conf, nil
}

func verifyConfig(c *Config) error {

	// TODO

	if c.ServerPort <= 0 || c.ServerPort > 65535 {
		return fmt.Errorf("invalid server port:%d", c.ServerPort)
	}

	return nil
}
