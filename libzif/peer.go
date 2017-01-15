// This represents a peer in the network.
// the minimum that a peer requires to be "valid" is just an address.
// everything else can be discovered via the network.
// Just a bit of a wrapper for the client really, that contains most of the networking code, this mostly has the data and a few other things.

package libzif

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/yamux"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ed25519"

	data "github.com/wjh/zif/libzif/data"
	"github.com/wjh/zif/libzif/dht"
	"github.com/wjh/zif/libzif/proto"
	"github.com/wjh/zif/libzif/util"
)

type Peer struct {
	address dht.Address

	publicKey ed25519.PublicKey
	streams   proto.StreamManager

	limiter *util.PeerLimiter

	entry *Entry

	// If this peer is acting as a seed for another
	seed    bool
	seedFor *Entry
}

func (p *Peer) Address() *dht.Address {
	return &p.address
}

func (p *Peer) PublicKey() []byte {
	return p.publicKey
}

func (p *Peer) Streams() *proto.StreamManager {
	return &p.streams
}

func (p *Peer) Announce(lp *LocalPeer) error {
	log.Debug("Sending announce to ", p.Address().String())

	if lp.Entry.PublicAddress == "" {
		log.Debug("Local peer public address is nil, attempting to fetch")
		ip := external_ip()
		log.Debug("External IP is ", ip)
		lp.Entry.PublicAddress = ip
	}
	lp.SignEntry()

	stream, err := p.OpenStream()

	if err != nil {
		return err
	}

	defer stream.Close()

	err = stream.Announce(lp.Entry)

	return err
}

func (p *Peer) Connect(addr string, lp *LocalPeer) error {
	log.Debug("Peer connecting to ", addr)
	pair, err := p.streams.OpenTCP(addr, lp)

	if err != nil {
		return err
	}

	p.publicKey = pair.PublicKey
	p.address = dht.NewAddress(pair.PublicKey)

	p.limiter = &util.PeerLimiter{}
	p.limiter.Setup()

	return nil
}

func (p *Peer) SetTCP(header proto.ConnHeader) {
	p.streams.SetConnection(header)

	p.publicKey = header.PublicKey
	p.address = dht.NewAddress(header.PublicKey)

	p.limiter = &util.PeerLimiter{}
	p.limiter.Setup()
}

func (p *Peer) ConnectServer() (*yamux.Session, error) {
	return p.streams.ConnectServer()
}

func (p *Peer) ConnectClient(lp *LocalPeer) (*yamux.Session, error) {
	client, err := p.streams.ConnectClient()

	if err != nil {
		return client, err
	}

	go lp.ListenStream(p)

	return client, err
}

func (p *Peer) Session() *yamux.Session {
	return p.streams.GetSession()
}

func (p *Peer) Terminate() {
	p.streams.Close()
}

func (p *Peer) OpenStream() (proto.Client, error) {
	if p.Session() == nil {
		return proto.Client{}, errors.New("Peer session nil")
	}

	if p.Session().IsClosed() {
		return proto.Client{}, errors.New("Peer session closed")
	}
	return p.streams.OpenStream()
}

func (p *Peer) AddStream(conn net.Conn) {
	p.streams.AddStream(conn)
}

func (p *Peer) RemoveStream(conn net.Conn) {
	p.streams.RemoveStream(conn)
}

func (p *Peer) GetStream(conn net.Conn) *proto.Client {
	return p.streams.GetStream(conn)
}

func (p *Peer) CloseStreams() {
	p.streams.Close()
}

func (p *Peer) Entry() (*Entry, error) {
	if p.entry != nil {
		return p.entry, nil
	}

	client, kv, err := p.Query(p.Address().String())

	if err != nil {
		return nil, err
	}

	defer client.Close()

	entry, err := JsonToEntry(kv.Value)

	if err != nil {
		return nil, err
	}

	log.WithField("addr", kv.Key.String()).Info("Recieved")

	if !entry.Address.Equals(p.Address()) {
		return nil, errors.New("Failed to fetch entry")
	}

	p.entry = entry

	return p.entry, nil
}

func (p *Peer) Ping() (time.Duration, error) {

	stream, err := p.OpenStream()
	defer stream.Close()

	if err != nil {
		log.Error(err.Error())
	}

	log.Info("Pinging ", p.Address().String())

	return stream.Ping(time.Second * 10)
}

func (p *Peer) Bootstrap(d *dht.DHT) (*proto.Client, error) {
	initial, err := p.Entry()

	if err != nil {
		return nil, err
	}

	dat, _ := initial.Json()

	d.Insert(dht.NewKeyValue(initial.Address, dat))

	stream, _ := p.OpenStream()

	return &stream, stream.Bootstrap(d, d.Address())
}

func (p *Peer) Query(address string) (*proto.Client, *dht.KeyValue, error) {
	log.WithField("target", address).Info("Querying")

	stream, _ := p.OpenStream()
	entry, err := stream.Query(address)
	return &stream, entry, err
}

func (p *Peer) FindClosest(address string) (*proto.Client, dht.Pairs, error) {
	log.WithField("target", address).Info("Finding closest")

	stream, _ := p.OpenStream()
	res, err := stream.FindClosest(address)
	return &stream, res, err
}

// asks a peer to query its database and return the results
func (p *Peer) Search(search string, page int) (*data.SearchResult, *proto.Client, error) {
	log.Info("Searching ", p.Address().String())
	stream, err := p.OpenStream()

	if err != nil {
		return nil, nil, err
	}

	posts, err := stream.Search(search, page)
	res := &data.SearchResult{
		Posts:  posts,
		Source: p.Address().String(),
	}

	if err != nil {
		return nil, nil, err
	}

	return res, &stream, nil
}

func (p *Peer) Recent(page int) ([]*data.Post, *proto.Client, error) {
	stream, err := p.OpenStream()

	if err != nil {
		return nil, nil, err
	}

	posts, err := stream.Recent(page)

	return posts, &stream, err

}

func (p *Peer) Popular(page int) ([]*data.Post, *proto.Client, error) {
	stream, err := p.OpenStream()

	if err != nil {
		return nil, nil, err
	}

	posts, err := stream.Popular(page)

	return posts, &stream, err

}

func (p *Peer) Mirror(db *data.Database, onPiece chan int) (*proto.Client, error) {
	pieces := make(chan *data.Piece, data.PieceSize)
	defer close(pieces)
	defer close(onPiece)

	go db.InsertPieces(pieces, true)

	log.WithField("peer", p.Address().String()).Info("Mirroring")

	stream, err := p.OpenStream()

	if err != nil {
		return nil, err
	}

	var entry *Entry
	if p.seed {
		entry = p.seedFor
	} else {
		entry, err = p.Entry()
	}

	if err != nil {
		return nil, err
	}

	mcol, err := stream.Collection(entry.Address, entry.PublicKey)

	if err != nil {
		return nil, err
	}

	collection := data.Collection{HashList: mcol.HashList}
	collection.Save(fmt.Sprintf("./data/%s/collection.dat", entry.Address.String()))

	if err != nil {
		return nil, err
	}

	log.Info("Downloading collection, size ", mcol.Size)

	piece_stream := stream.Pieces(entry.Address, 0, mcol.Size)

	i := 0
	for piece := range piece_stream {
		log.Info(len(mcol.HashList))
		hash := piece.Hash()

		if !bytes.Equal(mcol.HashList[32*i:32*i+32], hash) {
			return nil, errors.New("Piece hash mismatch")
		}

		onPiece <- i

		if len(pieces) == 100 {
			log.Info("Piece buffer full, io is blocking")
		}
		pieces <- piece

		i++
	}

	log.Info("Mirror complete")
	db.Close()

	p.RequestAddPeer(p.Address().String())

	return &stream, err
}

func (p *Peer) RequestAddPeer(addr string) (*proto.Client, error) {
	stream, err := p.OpenStream()

	if err != nil {
		return nil, err
	}

	return &stream, stream.RequestAddPeer(addr)
}
