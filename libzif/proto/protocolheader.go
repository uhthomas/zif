package proto

import (
	"github.com/zif/zif/libzif/dht"
	"golang.org/x/crypto/ed25519"
)

const ProtocolHeaderSize = 2 + 2 + ed25519.PublicKeySize

// Non-exported members are NOT sent over the wire.
// For instance, the zif address can be generated from the public key quite
// easily.
type ProtocolHeader struct {
	// This is Zif.
	Zif [2]byte

	// Protocol versions, ignores peers where this differs.
	Version [2]byte

	// Address from this, also used for verficication of other things.
	PublicKey [ed25519.PublicKeySize]byte

	address dht.Address
}
