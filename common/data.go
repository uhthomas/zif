package common

type Encodable interface {
	Bytes() ([]byte, error)
	String() (string, error)

	// The latter two may be equivelant
	Encode() ([]byte, error)
	EncodeString() (string, error)
}

type Signer interface {
	Sign([]byte) []byte
	PublicKey() []byte
}
