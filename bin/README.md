# bin
bin文件夹主要用于集中存储:
1. 各项配置文件、密钥文件等
2. 可执行文件
3. data文件夹存放数据库文件

在未编写好build.sh文件之前，
编译测试通过手动生成可执行文件进行

- 编译：
```
cd E:\GO\src\github.com\azd1997\ecoin\bin
go build -o ./ecli.exe ../cmd/ecli
go build -o ./ecoind.exe ../cmd/ecoind
```

- 创建账户
    - 默认账户文件路径："./account.json"
```
# 在bin目录下生成一些测试账号
ecli new account -r 1 -o ./test/acc1.json
```

现在我总共生成了6个角色类型为1的账户，分别是：
```json
[{
  "_comment": "acc0. main account. genesis",
  "roleNo": 1,
  "privKeyB": "Ahvgc0UpOJiKrT8M2A/XBw8c/Saen59HDK/EH+wyuCI=",
  "node addr": ":7000"
},
{
  "_comment": "acc1",
  "roleNo": 1,
  "privKeyB": "7MPzMiR3aXokic6GvYLm9zDU7aQsTLawO7Zl/rIvf7w=",
  "node addr": ":7001"
},
{
  "_comment": "acc2",
  "roleNo": 1,
  "privKeyB": "2YFm0fPInc5iZU17jSElq289sSSNhhKuiCii6ruCpOw=",
  "node addr": ":7002"
},
{
  "_comment": "acc3",
  "roleNo": 1,
  "privKeyB": "yvtbgpgAAAGAOpal/0nFLr+fDSCi+2cDu9uDA0JZVzs=",
  "node addr": ":7003"  
},
{
  "_comment": "acc4",
  "roleNo": 1,
  "privKeyB": "n0j52YhBrzmktsSSDYki+D7Hw6sNMuAhtSOjil5Qx2E=",
  "node addr": ":7004"  
},
{
  "_comment": "acc5",
  "roleNo": 1,
  "privKeyB": "djkKh3wSr+D31b/SwVm2W0ngmcvsr7LH6gisOdcfDWU=",
  "node addr": ":7005"  
}]
```

测试流程：
1. 编译二进制文件
2. 准备好多个账户
3. 选择某个账户(acc0)构建genesis交易，并将其加入到所有测试节点配置文件中
4. 种子节点列表填写。genesis账户(acc0)对应的节点。填到acc1-5账户节点的配置文件中
5. 启动acc0对应节点，由于网络中目前只有一个节点，节点0不断产生空区块
6. 接着顺次启动节点1-5
7. 启动节点1时，会向种子节点请求节点列表，这时双方都有列表{node0, node1}
8. 启动节点2-5类似
9. 转账：节点0有创世奖励，拥有余额，节点0向节点1发起一笔普通转账交易
10. 下一个区块中应当是包含这个交易的