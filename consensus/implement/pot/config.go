/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/8/26 18:12
* @Description: The file is for
***********************************************************************/

package pot

import "time"

// Config PoT节点配置项
type Config struct {
	// ID 节点的唯一标识
	ID string

	// peers 维护的所有其他节点的ID
	peers []string

	Storage Storage

	// Timer 定时器
	Timer time.Timer
}

func (c *Config) validate() error {
	return nil
}