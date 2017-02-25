// a few network helpers
package proto

import "github.com/zif/zif/dht"

type ConnHeader struct {
	Client       Client
	Entry        dht.Entry
	Capabilities MessageCapabilities
}
