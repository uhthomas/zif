package proto

import (
	"net"

	"github.com/hashicorp/yamux"
	"github.com/zif/zif/common"
	"github.com/zif/zif/dht"
)

type ProtocolHandler interface {
	common.Signer
	NetworkPeer

	HandleAnnounce(*Message) error
	HandleQuery(*Message) error
	HandleFindClosest(*Message) error
	HandleSearch(*Message) error
	HandleRecent(*Message) error
	HandlePopular(*Message) error
	HandleHashList(*Message) error
	HandlePiece(*Message) error
	HandleAddPeer(*Message) error

	HandleHandshake(ConnHeader) (NetworkPeer, error)
	HandleCloseConnection(*dht.Address)

	GetNetworkPeer(string) NetworkPeer
	SetNetworkPeer(NetworkPeer)
}

// Allows the protocol stuff to work with Peers, while libzif/peer can interface
// peers with the DHT properly.
type NetworkPeer interface {
	Session() *yamux.Session
	AddStream(net.Conn)

	Address() *dht.Address
	Query(string) (common.Verifiable, error)
	FindClosest(address string) ([]common.Verifiable, error)
}
