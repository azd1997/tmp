package erro

import "errors"

var (
	ErrSendToSelf = errors.New("send to self")
	ErrInvalidAddr = errors.New("invalid addr")
	ErrUnreachableNode = errors.New("unreachable node")
)
var (
	ErrDbNotExists = errors.New("database not exists")
)