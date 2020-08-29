/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/8/26 18:47
* @Description: 存储（也就是区块链）的状态定义，包括链状态和集群配置状态
***********************************************************************/

package pot

type HardState struct {

}

type ConfState struct {
	Nodes []string
}