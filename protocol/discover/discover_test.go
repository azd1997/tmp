package discover

import (
	"bytes"
	"testing"

	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/utils"
)

func verifyHead(t *testing.T, expect *Head, result *Head) {
	if err := utils.TCheckUint8("head version", expect.Version, result.Version); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckInt64("head time", expect.Time, result.Time); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint8("head type", expect.Type, result.Type); err != nil {
		t.Fatal(err)
	}
}

func TestPing(t *testing.T) {
	from := crypto.RandID()
	//fmt.Println(len(from))

	ping := NewPingMsg(from)
	pingBytes := ping.Encode()

	rPing := &PingMsg{}
	err := rPing.Decode(bytes.NewReader(pingBytes))
	if err != nil {
		t.Fatalf("decode PingMsg failed: %v\n", err)
	}

	// verify
	verifyHead(t, ping.Head, rPing.Head)

	if err := utils.TCheckString("from id", string(from), string(rPing.From)); err != nil {
		t.Fatal(err)
	}
}

func TestPong(t *testing.T) {
	hash := crypto.RandHash()
	from := crypto.RandID()

	pong := NewPongMsg(hash, from)
	pongBytes := pong.Encode()

	rPong := &PongMsg{}
	err := rPong.Decode(bytes.NewReader(pongBytes))
	if err != nil {
		t.Fatalf("decode PongMsg failed: %v\n", err)
	}

	verifyHead(t, pong.Head, rPong.Head)
	if err := utils.TCheckBytes("ping hash", pong.PingHash, rPong.PingHash); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckString("from id", string(pong.From), string(rPong.From)); err != nil {
		t.Fatal(err)
	}
}

func TestGetNeighbours(t *testing.T) {
	from := crypto.RandID()

	getNeighbours := NewGetNeighboursMsg(from)
	getNeighboursBytes := getNeighbours.Encode()

	rGetNeighbours := &GetNeighboursMsg{}
	err := rGetNeighbours.Decode(bytes.NewReader(getNeighboursBytes))
	if err != nil {
		t.Fatalf("decode GetNeighboursMsg failed: %v\n", err)
	}

	verifyHead(t, getNeighbours.Head, rGetNeighbours.Head)
	if err := utils.TCheckString("from id", string(getNeighbours.From), string(rGetNeighbours.From)); err != nil {
		t.Fatal(err)
	}
}

func TestNeighbours(t *testing.T) {
	from := crypto.RandID()

	// empty
	emptyNeighbour := NewNeighboursMsg(from, nil)
	emptyNeighbourBytes := emptyNeighbour.Encode()

	rEmptyNeighbour := &NeighboursMsg{}
	err := rEmptyNeighbour.Decode(bytes.NewReader(emptyNeighbourBytes))
	if err != nil {
		t.Fatalf("decode empty NeighboursMsg failed: %v\n", err)
	}
	verifyHead(t, emptyNeighbour.Head, rEmptyNeighbour.Head)

	if err := utils.TCheckInt("nodes number", 0, len(rEmptyNeighbour.Nodes)); err != nil {
		t.Fatal(err)
	}

	// with 2 nodes
	nodes := []*Node{
		NewNode(NewAddress("8.8.8.8", int32(10000)), crypto.RandID()),
		NewNode(NewAddress("6.6.6.6", int32(10080)), crypto.RandID()),
	}
	neighbours := NewNeighboursMsg(from, nodes)
	neighboursBytes := neighbours.Encode()
	rNeighbours := &NeighboursMsg{}
	err = rNeighbours.Decode(bytes.NewReader(neighboursBytes))
	if err != nil {
		t.Fatalf("decode NeighboursMsg failed: %v\n", err)
	}
	verifyHead(t, neighbours.Head, rNeighbours.Head)

	for i, node := range rNeighbours.Nodes {
		if err := utils.TCheckIP("neighbour ip", nodes[i].Addr.IP, node.Addr.IP); err != nil {
			t.Fatal(err)
		}
		if err := utils.TCheckInt32("neighbour port", nodes[i].Addr.Port, node.Addr.Port); err != nil {
			t.Fatal(err)
		}
		if err := utils.TCheckString("neighbour node id", string(nodes[i].ID), string(node.ID)); err != nil {
			t.Fatal(err)
		}
	}
}

