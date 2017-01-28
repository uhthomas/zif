package common

import "github.com/zif/zif/dht"

type ConnectPeer func(dht.Address) (interface{}, error)

type Peer interface {
	EAddress() Encoder
	FindClosest(dht.Address) ([]Verifier, error)
	Query(dht.Address) (Verifier, error)
}

type Closable interface {
	Close() error
}
