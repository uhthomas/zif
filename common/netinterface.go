package common

import "github.com/zif/zif/dht"

type ConnectPeer func(string) (interface{}, error)

type Peer interface {
	EAddress() Encodable
	FindClosest(dht.Address) ([]Verifiable, error)
	Query(dht.Address) (Verifiable, error)
}

type Closable interface {
	Close() error
}
