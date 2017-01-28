package proto

import (
	"bytes"
	"compress/gzip"
	"io"
	"net"

	msgpack "gopkg.in/vmihailenco/msgpack.v2"

	"github.com/zif/zif/common"
	"github.com/zif/zif/dht"
)

type Message struct {
	Header string
	Stream net.Conn
	Client *Client
	From   *dht.Address

	content bytes.Buffer
}

func (m *Message) Content() *bytes.Buffer {
	return &m.content
}

func (m *Message) ContentLength() int {
	return m.content.Len()
}

func (m *Message) Write(iface interface{}) error {
	compressor := gzip.NewWriter(&m.content)
	encoder := msgpack.NewEncoder(compressor)

	err := encoder.Encode(iface)

	if err != nil {
		return err
	}

	err = compressor.Flush()

	if err != nil {
		return err
	}

	return nil
}

func (m *Message) Read(iface interface{}) error {
	decompressor, err := gzip.NewReader(&m.content)
	limiter := &io.LimitedReader{decompressor, common.MaxMessageContentSize}

	if err != nil {
		return err
	}

	decoder := msgpack.NewDecoder(limiter)

	err = decoder.Decode(iface)

	return err
}

func (m *Message) ReadInt() (int, error) {
	var ret int

	err := m.Read(&ret)

	return ret, err
}

func (m *Message) Json() ([]byte, error) {
	return msgpack.Marshal(m)
}

// Ok() is just an easier way to check if the peer has sent an "ok" response,
// rather than comparing the header member to a constant repeatedly.
func (m *Message) Ok() bool {
	return m.Header == ProtoOk
}
