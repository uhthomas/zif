package common

type ConnectPeer func(string) (interface{}, error)

type Peer interface {
	EAddress() Encodable
	FindClosest(address string) ([]Verifiable, error)
}

type Closable interface {
	Close() error
}
