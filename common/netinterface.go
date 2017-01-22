package common

import "github.com/zif/zif/dht"

type ConnectPeer func(string) (interface{}, error)

type Peer interface {
	EAddress() Encodable
	Query(string) (*dht.KeyValue, error)
	FindClosest(address string) (dht.Pairs, error)
}

type Closable interface {
	Close() error
}
