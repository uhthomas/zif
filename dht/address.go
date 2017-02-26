// Generates a zif address given a public key
// Similar to the method Bitcoin uses
// see: https://en.bitcoin.it/wiki/Technical_background_of_version_1_Bitcoin_addresses

package dht

import (
	"bytes"
	"encoding/json"
	"errors"

	msgpack "gopkg.in/vmihailenco/msgpack.v2"

	"github.com/codahale/blake2"
	"github.com/wjh/hellobitcoin/base58check"
	"github.com/zif/zif/util"
	"golang.org/x/crypto/sha3"
)

const AddressBinarySize = 20
const AddressVersion = 0

type Address struct {
	Raw []byte
}

// Generates an Address from a PublicKey.
func NewAddress(key []byte) (addr Address) {
	addr = Address{}
	addr.Generate(key)

	return
}

// Returns Address.Bytes Base58 encoded and prepended with a Z.
// Base58 removes ambiguous characters, reducing the chances of address confusion.
func (a Address) String() (string, error) {
	b, _ := a.Bytes()
	return base58check.Encode("51", b)
}

func (a Address) StringOr(or string) string {
	str, err := a.String()

	if err != nil {
		return or
	}

	return str
}

func (a *Address) Bytes() ([]byte, error) {
	return a.Raw, nil
}

func (a *Address) Encode() ([]byte, error) {
	return msgpack.Marshal(a)
}

func (a *Address) EncodeString() (string, error) {
	dat, err := json.Marshal(a)

	return string(dat), err
}

// Decodes a string address into address bytes.
func DecodeAddress(value string) (Address, error) {
	var addr Address
	var err error
	addr.Raw, err = base58check.Decode(value)

	return addr, err
}

func RandomAddress() (*Address, error) {
	rand, err := util.CryptoRandBytes(32)

	if err != nil {
		return nil, err
	}

	addr := Address{}
	_, err = addr.Generate(rand)

	return &addr, err
}

// Generate a Zif address from a public key.
// This process involves one SHA3-256 iteration, followed by BLAKE2. This is
// similar to bitcoin, and the BLAKE2 makes the resulting address a bit shorter
func (a *Address) Generate(key []byte) (string, error) {
	blake := blake2.New(&blake2.Config{Size: AddressBinarySize})

	if len(key) != 32 {
		return "", (errors.New("Public key is not 32 bytes"))
	}

	// Why hash and not just use the pub key?
	// This way we can change curve or algorithm entirely, and still have
	// the same format for addresses.

	firstHash := sha3.Sum256(key)
	blake.Write(firstHash[:])

	secondHash := blake.Sum(nil)

	a.Raw = secondHash

	s, _ := a.String()
	return s, nil
}

func (a *Address) Less(other *Address) bool {

	for i := 0; i < len(a.Raw); i++ {
		if a.Raw[i] != other.Raw[i] {
			return a.Raw[i] < other.Raw[i]
		}
	}

	return false
}

func (a *Address) Xor(other *Address) *Address {
	var ret Address
	ret.Raw = make([]byte, len(a.Raw))

	for i := 0; i < len(a.Raw); i++ {
		ret.Raw[i] = a.Raw[i] ^ other.Raw[i]
	}

	return &ret
}

// Counts the number of leading zeroes this address has.
// The address should be the result of an Xor.
// This shows the k-bucket that this address should go into.
func (a *Address) LeadingZeroes() int {

	for i := 0; i < len(a.Raw); i++ {
		for j := 0; j < 8; j++ {
			if (a.Raw[i]>>uint8(7-j))&0x1 != 0 {
				return i*8 + j
			}
		}
	}

	return len(a.Raw)*8 - 1
}

func (a *Address) Equals(other *Address) bool {
	return bytes.Equal(a.Raw, other.Raw)
}
