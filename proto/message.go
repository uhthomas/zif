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

	Content []byte
}

func (m *Message) Write(iface interface{}) error {
	writer := bytes.Buffer{}
	compressor := gzip.NewWriter(&writer)
	encoder := msgpack.NewEncoder(compressor)

	err := encoder.Encode(iface)

	if err != nil {
		return err
	}

	err = compressor.Flush()

	if err != nil {
		return err
	}

	m.Content = writer.Bytes()

	return nil
}

func (m *Message) Read(iface interface{}) error {
	reader := bytes.NewReader(m.Content)
	decompressor, err := gzip.NewReader(reader)
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
