package p2p

import (
	"errors"
	"fmt"

	"github.com/azd1997/ecoin/common/params"
)

var ErrNegotiateInvalidSig = errors.New("invalid signature")

var ErrNegotiateConnectionRefused = errors.New("connection refused")

var ErrNegotiateChainIDMismatch = errors.New("chain id mismatch")

var ErrNegotiateNodeRoleMismatch = errors.New("node role mismatch: the rules about communication among different roles TODO")

var ErrNegotiateTimeout = errors.New("timeout")

type ErrNegotiateCodeVersionMismatch struct {
	minimizeVersionRequired params.CodeVersion
	remoteVersion           params.CodeVersion
}

func (n ErrNegotiateCodeVersionMismatch) Error() string {
	return fmt.Sprintf("code version mismatch, minimize required %d, got %d",
		n.minimizeVersionRequired, n.remoteVersion)
}

type ErrNegotiateBrokenData struct {
	info string
}

func (n ErrNegotiateBrokenData) Error() string {
	return n.info
}
