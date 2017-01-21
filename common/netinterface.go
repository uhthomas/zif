package common

import "github.com/zif/zif/dht"

type ConnectPeer func(string) (interface{}, error)

type Peer interface {
	EAddress() Encodable
	Query(string) (Closable, *dht.KeyValue, error)
	FindClosest(address string) (Closable, dht.Pairs, error)
}

type Closable interface {
	Close() error
}
