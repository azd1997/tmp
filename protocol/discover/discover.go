package discover

type DiscoverMsgType = uint8

const (
    // DiscoverV1 is the version 1 of the discover protocol
    DISCOVER_V1 = 1

    // discover message type
    MSG_PING          = DiscoverMsgType(1)
    MSG_PONG         = DiscoverMsgType(2)
    MSG_GET_NEIGHBERS = DiscoverMsgType(3)
    MSG_NEIGHBERS    = DiscoverMsgType(4)
)
