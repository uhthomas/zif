package proto

import (
	"net"

	"github.com/hashicorp/yamux"
	"github.com/zif/zif/libzif/common"
	"github.com/zif/zif/libzif/dht"
)

type ProtocolHandler interface {
	common.Signer

	HandleAnnounce(*Message) error
	HandleQuery(*Message) error
	HandleFindClosest(*Message) error
	HandleSearch(*Message) error
	HandleRecent(*Message) error
	HandlePopular(*Message) error
	HandleHashList(*Message) error
	HandlePiece(*Message) error
	HandleAddPeer(*Message) error
	HandlePing(*Message) error

	HandleHandshake(ConnHeader) (NetworkPeer, error)
	HandleCloseConnection(*dht.Address)
}

// Allows the protocol stuff to work with Peers, while libzif/peer can interface
// peers with the DHT properly.
type NetworkPeer interface {
	Session() *yamux.Session
	AddStream(net.Conn)

	Address() *dht.Address
	Query(string) (*Client, *dht.KeyValue, error)
	FindClosest(address string) (*Client, dht.Pairs, error)
}
