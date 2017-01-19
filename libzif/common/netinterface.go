package common

type ConnectPeer func(string) (Peer, error)

type Peer interface {
	Address() *Encodable
	Query(string) (Client, KeyValue, error)
	FindClosest(address string) (Client, Pairs, error)
}

type Client interface {
	Close() error
}

type KeyValue interface {
	Key() *Encodable
	Value() []byte
}

type Pairs []*KeyValue
