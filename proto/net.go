// a few network helpers

package proto

import "golang.org/x/crypto/ed25519"

type ConnHeader struct {
	Client    Client
	PublicKey ed25519.PublicKey
}
