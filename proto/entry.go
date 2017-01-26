package proto

import (
	"encoding/json"
	"errors"
	"fmt"

	msgpack "gopkg.in/vmihailenco/msgpack.v2"

	"github.com/zif/zif/dht"

	"golang.org/x/crypto/ed25519"
)

// This is an entry into the DHT. It is used to connect to a peer given just
// it's Zif address.
type Entry struct {
	Address       dht.Address `json:"address"`
	Name          string      `json:"name"`
	Desc          string      `json:"desc"`
	PublicAddress string      `json:"publicAddress"`
	PublicKey     []byte      `json:"publicKey"`
	PostCount     int         `json:"postCount"`
	Updated       uint64      `json:"updated"`

	// The owner of this entry should have signed it, we need to store the
	// sigature. It's actually okay as we can verify that a peer owns a public
	// key by generating an address from it - if the address is not the peers,
	// then Mallory is just using someone elses entry for their own address.
	Signature []byte `json:"signature"`
	// Signature of the root hash of a hash list representing all of the posts
	// a peer has.
	CollectionSig  []byte `json:"collectionSig"`
	CollectionHash []byte `json:"collectionHash"`
	Port           int    `json:"port"`

	// Essentially just a list of other peers who have this entry in their table.
	// They may or may not actually have pieces, so mirror/piece requests may go
	// awry.
	// This is... weird. It is not signed. It is not verified.
	// While the idea of doing the above irks me somewhat, as potentially bad
	// actors can make themselves become seeds - they will fail with piece
	// requests. Hashes of pieces will not match.
	// Removing the requirement for this to be both signed and verified means
	// that any peer can become a seed without that much work at all. Again,
	// seed lists can then be updated *without* the requirement that the origin
	// peer actually be online in the first place.
	// TODO: Switch this to be a struct containing the last time this peer was
	// announced as a peer, then the list can be periodically culled.
	Seeds [][]byte `json:"seeds"`

	// Used in the FindClosest function, for sorting.
	distance dht.Address
}

// true if JSON, false if msgpack
func DecodeEntry(data []byte, isJson bool) (*Entry, error) {
	var err error
	e := &Entry{}

	if isJson {
		err = json.Unmarshal(data, e)
	} else {
		err = msgpack.Unmarshal(data, e)
	}

	if err != nil {
		return nil, err
	}

	return e, nil
}

// This is signed, *not* the JSON. This is needed because otherwise the order of
// the posts encoded is not actually guaranteed, which can lead to invalid
// signatures. Plus we can only sign data that is actually needed.
func (e Entry) Bytes() ([]byte, error) {
	ret, err := e.String()
	return []byte(ret), err
}

func (e Entry) String() (string, error) {
	var str string

	str += e.Name
	str += e.Desc
	str += string(e.PublicKey)
	str += string(e.Port)
	str += string(e.PublicAddress)
	s, _ := e.Address.String()
	str += s
	str += string(e.PostCount)

	return str, nil
}

func (e Entry) Encode() ([]byte, error) {
	return msgpack.Marshal(e)
}

// Returns a JSON encoded string, not msgpack. This is because it is likely
// going to be seen by a human, otherwise it would be bytes.
func (e Entry) EncodeString() (string, error) {
	enc, err := json.Marshal(e)

	if err != nil {
		return "", err
	}

	return string(enc), err
}

func (e *Entry) SetLocalPeer(lp ProtocolHandler) {
	e.Address = *lp.Address()

	e.PublicKey = make([]byte, len(lp.PublicKey()))
	copy(e.PublicKey, lp.PublicKey())
	e.PublicKey = lp.PublicKey()
}

type Entries []*Entry

func (e Entries) Len() int {
	return len(e)
}

func (e Entries) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e Entries) Less(i, j int) bool {
	return e[i].distance.Less(&e[j].distance)
}

// Ensures that all the members of an entry struct fit the requirements for the
// Zif libzifcol. If an entry passes this, then we should be able to perform
// most operations on it.
func (entry *Entry) Verify() error {
	if len(entry.PublicKey) < ed25519.PublicKeySize {
		return errors.New(fmt.Sprintf("Public key too small: %d", len(entry.PublicKey)))
	}

	if len(entry.Signature) < ed25519.SignatureSize {
		return errors.New("Signature too small")
	}

	data, _ := entry.Bytes()
	verified := ed25519.Verify(entry.PublicKey, data, entry.Signature[:])

	if !verified {
		return errors.New("Failed to verify signature")
	}

	if len(entry.PublicAddress) == 0 {
		return errors.New("Public address must be set")
	}

	// 253 is the maximum length of a domain name
	if len(entry.PublicAddress) >= 253 {
		return errors.New("Public address is too large (253 char max)")
	}

	if entry.Port > 65535 {
		return errors.New("Port too large (" + string(entry.Port) + ")")
	}

	return nil
}
