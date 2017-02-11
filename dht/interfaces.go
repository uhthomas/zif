package dht

type Node interface {
	Address() *Address
	PublicKey() []byte
}
