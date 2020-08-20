package p2p

import (
	"github.com/azd1997/ego/epattern"

	"github.com/azd1997/ecoin/p2p/peer"
)

type recvHandler = func(peer peer.ID, protocolID uint8, data []byte)

// conn 代表了本机节点node与对方节点peer之间通信的链接，包括通信双方（node, p），TCP链接conn，会话加解密模块ec
// 接收事务处理器（决定接收到某种消息时执行什么任务）handler， lm循环工作模式
type conn struct {
	node    *Node
	p       *peer.Peer
	conn    TCPConn
	ec      codec
	handler recvHandler
	lm      *epattern.LoopMode
}

func newConn(p *peer.Peer, nc TCPConn, ec codec, handler recvHandler) *conn {
	c := &conn{
		p:       p,
		conn:    nc,
		ec:      ec,
		handler: handler,
		lm:      epattern.NewLoop(1),	// 每一个链接只会起一个go程用于循环处理事务
	}

	return c
}

// 启动链接工作循环
func (c *conn) start() {
	go c.loop()
	c.lm.StartWorking()
}

// 停止链接工作循环，如果工作循环正在运行则关闭它并随后关闭真正的通信链接c.conn
func (c *conn) stop() {
	if c.lm.Stop() {
		c.conn.Disconnect()
	}
}

// 工作循环。 接收数据包并相应处理
func (c *conn) loop() {
	c.lm.Add()	// 添加当前goroutine，等待结束
	defer c.lm.Done()

	// TCP链接 数据packet接收通道
	recvC := c.conn.GetRecvChannel()
	for {
		select {
		case <-c.lm.D:
			return
		case pkt := <-recvC:
			var ok bool
			var payload []byte
			var protocolID uint8
			// 验证packet是否合法、完整
			if ok, payload, protocolID = verifyTCPPacket(pkt); !ok {
				logger.Warnln("verify packet failed, close connection")
				go c.stop()
				break
			}
			// 将payload解密成明文
			plaintext, err := c.ec.decrypt(payload)
			if err != nil {
				logger.Warnln("decrypt packet failed, close connection")
				go c.stop()
				break
			}
			// 处理当前payload
			c.handler(c.p.ID, protocolID, plaintext)
		}
	}
}

// 发送数据。 参数data即为payload明文
func (c *conn) send(protocolID uint8, data []byte) {
	// 加密
	cipherText, err := c.ec.encrypt(data)
	if err != nil {
		logger.Warn("encrypt payload failed, close connection")
		go c.stop()
		return
	}
	// 构建TCP packet
	pkt := buildTCPPacket(cipherText, protocolID)
	c.conn.Send(pkt)
}
