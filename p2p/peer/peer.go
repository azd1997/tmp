package peer

import (
	"fmt"
	"net"

	"github.com/azd1997/ecoin/account"
)

// Peer 代表了一个可连接的节点
type Peer struct {
	IP   net.IP
	Port int

	// 将账户与节点绑定起来
	// ID将作为Peer的另一种标识
	ID account.UserId
}

// NewPeer 新建一个节点标识
func NewPeer(ip net.IP, port int, id account.UserId) *Peer {
	p := &Peer{
		IP:   ip,
		Port: port,
		ID:id,
	}
	return p
}

func (p *Peer) String() string {
	return fmt.Sprintf("ID %s address %s", p.ID.Id, p.Address())
}

// Address 返回节点IP地址，例如 192.168.1.1:8080,[2001:0db8:85a3:08d3:1319:8a2e:0370:7344]:8443
// 支持IPV4及IPV6
func (p *Peer) Address() string {
	v4IP := p.IP.To4()
	if v4IP != nil {
		return fmt.Sprintf("%s:%d", v4IP.String(), p.Port)
	}
	return fmt.Sprintf("[%s]:%d", p.IP.String(), p.Port)

}