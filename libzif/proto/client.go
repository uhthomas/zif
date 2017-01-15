package proto

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"time"

	"golang.org/x/crypto/ed25519"

	log "github.com/sirupsen/logrus"
	"github.com/wjh/zif/libzif/data"
	"github.com/wjh/zif/libzif/dht"
)

const (
	EntryLengthMax = 1024
	MaxPageSize    = 25
)

type Client struct {
	conn net.Conn

	decoder *json.Decoder
	encoder *json.Encoder
}

// Creates a new client, automatically setting up the json encoder/decoder.
func NewClient(conn net.Conn) *Client {
	return &Client{conn, json.NewDecoder(conn), json.NewEncoder(conn)}
}

func (c *Client) Terminate() {
	//c.conn.Write(proto_terminate)
}

// Close the client connection.
func (c *Client) Close() (err error) {
	if c.conn != nil {
		err = c.conn.Close()
	}
	return
}

// Encodes v as json and writes it to c.conn.
func (c *Client) WriteMessage(v interface{}) error {
	if c.encoder == nil {
		c.encoder = json.NewEncoder(c.conn)
	}

	err := c.encoder.Encode(v)

	return err
}

// Blocks until a message is read from c.conn, decodes it into a *Message and
// returns.
func (c *Client) ReadMessage() (*Message, error) {
	var msg Message

	if c.decoder == nil {
		c.decoder = json.NewDecoder(c.conn)
	}

	if err := c.decoder.Decode(&msg); err != nil {
		return nil, err
	}

	msg.Stream = c.conn

	return &msg, nil
}

func (c *Client) Decode(i interface{}) error {
	return c.decoder.Decode(i)
}

// Pings a client with a specified timeout, returns true/false depending on
// if it recieves a reply.
func (c *Client) Ping(timeout time.Duration) (time.Duration, error) {
	start := time.Now()

	c.WriteMessage(&Message{Header: ProtoPing})

	tchan := make(chan bool)

	go func() {
		rep, err := c.ReadMessage()

		if err != nil || rep.Header != ProtoPong {
			tchan <- false
		}

		tchan <- true
	}()

	select {
	case <-tchan:
		return time.Since(start), nil
	case <-time.After(timeout):
		return time.Since(start), errors.New("Ping timeout")
	}
}

// Replies to a Ping request.
func (c *Client) Pong() {
	//c.conn.Write(proto_pong)
}

// Sends a DHT entry to a peer.
func (c *Client) SendStruct(e data.Encodable) error {
	json, err := e.Json()
	msg := Message{Header: ProtoEntry, Content: json}

	if err != nil {
		c.conn.Close()
		return err
	}

	c.WriteMessage(msg)

	return nil
}

// Announce the given DHT entry to a peer, passes on this peers details,
// meaning that it can be reached by other peers on the network.
func (c *Client) Announce(e data.Encodable) error {
	json, err := e.Json()

	if err != nil {
		c.conn.Close()
		return err
	}

	msg := &Message{
		Header:  ProtoDhtAnnounce,
		Content: json,
	}

	err = c.WriteMessage(msg)

	if err != nil {
		return err
	}

	ok, err := c.ReadMessage()

	if err != nil {
		return err
	}

	if !ok.Ok() {
		return errors.New("Peer did not respond with ok")
	}

	return nil
}

func (c *Client) FindClosest(address string) (dht.Pairs, error) {
	// TODO: LimitReader

	msg := &Message{
		Header:  ProtoDhtFindClosest,
		Content: []byte(address),
	}

	// Tell the peer the address we are looking for
	err := c.WriteMessage(msg)

	if err != nil {
		return nil, err
	}

	log.Debug("Send FindClosest request")

	// Make sure the peer accepts the address
	recv, err := c.ReadMessage()

	if err != nil {
		return nil, err
	}

	log.Debug("Peer accepted address")

	if !recv.Ok() {
		return nil, errors.New("Peer refused query address")
	}

	closest, err := c.ReadMessage()

	if err != nil {
		return nil, err
	}

	length, err := closest.ReadInt()

	if err != nil {
		return nil, err
	}

	entries := make(dht.Pairs, 0, length)

	for i := 0; i < length; i++ {
		kv := &dht.KeyValue{}
		err = c.Decode(kv)

		if err != nil {
			return nil, err
		}

		entries = append(entries, kv)
	}

	log.WithField("entries", len(entries)).Info("Find closest complete")
	return entries, err
}

func (c *Client) Query(address string) (*dht.KeyValue, error) {
	// TODO: LimitReader

	msg := &Message{
		Header:  ProtoDhtQuery,
		Content: []byte(address),
	}

	// Tell the peer the address we are looking for
	err := c.WriteMessage(msg)

	if err != nil {
		return nil, err
	}

	// Make sure the peer accepts the address
	recv, err := c.ReadMessage()

	if err != nil {
		return nil, err
	}

	if !recv.Ok() {
		return nil, errors.New("Peer refused query address")
	}

	kv := &dht.KeyValue{}
	err = c.Decode(kv)

	return kv, err

}

// Adds the initial entries into the given routing table. Essentially queries for
// both it's own and the peers address, storing the result. This means that after
// a bootstrap, it should be possible to connect to *any* peer!
func (c *Client) Bootstrap(d *dht.DHT, address dht.Address) error {
	peers, err := c.FindClosest(address.String())

	if err != nil {
		return err
	}

	// add them all to our routing table! :D
	for _, e := range peers {
		if len(e.Key.Raw) != dht.AddressBinarySize {
			continue
		}

		d.Insert(e)
	}

	if len(peers) > 1 {
		log.Info("Bootstrapped with ", len(peers), " new peers")
	} else if len(peers) == 1 {
		log.Info("Bootstrapped with 1 new peer")
	}

	return nil
}

// TODO: Paginate searches
func (c *Client) Search(search string, page int) ([]*data.Post, error) {
	log.Info("Querying for ", search)

	sq := MessageSearchQuery{search, page}
	dat, err := sq.Encode()

	if err != nil {
		return nil, err
	}

	msg := &Message{
		Header:  ProtoSearch,
		Content: dat,
	}

	c.WriteMessage(msg)

	var posts []*data.Post

	recv, err := c.ReadMessage()

	if err != nil {
		return nil, err
	}

	err = recv.Decode(&posts)

	if err != nil {
		return nil, err
	}

	return posts, nil
}

func (c *Client) Recent(page int) ([]*data.Post, error) {
	log.Info("Fetching recent posts from peer")

	page_s := strconv.Itoa(page)

	msg := &Message{
		Header:  ProtoRecent,
		Content: []byte(page_s),
	}

	err := c.WriteMessage(msg)

	if err != nil {
		return nil, err
	}

	posts_msg, err := c.ReadMessage()

	if err != nil {
		return nil, err
	}

	var posts []*data.Post
	posts_msg.Decode(&posts)

	log.Info("Recieved ", len(posts), " recent posts")

	return posts, nil
}

func (c *Client) Popular(page int) ([]*data.Post, error) {
	log.Info("Fetching popular posts from peer")

	page_s := strconv.Itoa(page)

	msg := &Message{
		Header:  ProtoPopular,
		Content: []byte(page_s),
	}

	err := c.WriteMessage(msg)

	if err != nil {
		return nil, err
	}

	posts_msg, err := c.ReadMessage()

	if err != nil {
		return nil, err
	}

	var posts []*data.Post
	posts_msg.Decode(&posts)

	log.Info("Recieved ", len(posts), " popular posts")

	return posts, nil
}

// Download a hash list for a peer. Expects said hash list to be valid and
// signed.
func (c *Client) Collection(address dht.Address, pk ed25519.PublicKey) (*MessageCollection, error) {
	log.WithField("for", address.String()).Info("Sending request for a collection")

	msg := &Message{
		Header:  ProtoRequestHashList,
		Content: address.Bytes(),
	}

	c.WriteMessage(msg)

	hl, err := c.ReadMessage()

	if err != nil {
		return nil, err
	}

	mhl := MessageCollection{}
	err = hl.Decode(&mhl)

	if err != nil {
		return nil, err
	}

	err = mhl.Verify(pk)

	if err != nil {
		return nil, err
	}

	log.WithField("pieces", mhl.Size).Info("Recieved valid collection")

	return &mhl, nil
}

// Download a piece from a peer, given the address and id of the piece we want.
func (c *Client) Pieces(address dht.Address, id, length int) chan *data.Piece {
	log.WithFields(log.Fields{
		"address": address.String(),
		"id":      id,
		"length":  length,
	}).Info("Sending request for piece")

	ret := make(chan *data.Piece, 100)

	mrp := MessageRequestPiece{address.String(), id, length}
	dat, err := mrp.Encode()

	if err != nil {
		return nil
	}

	msg := &Message{
		Header:  ProtoRequestPiece,
		Content: dat,
	}

	c.WriteMessage(msg)

	// Convert a string to an int, prevents endless error checks below.
	convert := func(val string) int {
		ret, err := strconv.Atoi(val)

		if err != nil {
			log.Error(err.Error())
			return 0
		}

		return ret
	}

	go func() {
		defer close(ret)

		gzr, err := gzip.NewReader(c.conn)

		if err != nil {
			return
		}

		errReader := data.NewErrorReader(gzr)

		for i := 0; i < length; i++ {
			piece := data.Piece{}
			piece.Setup()

			count := 0
			for {
				if count >= data.PieceSize {
					break
				}

				id_s := errReader.ReadString('|')
				id := convert(id_s)

				if id == -1 {
					break
				}

				ih := errReader.ReadString('|')
				title := errReader.ReadString('|')
				size := convert(errReader.ReadString('|'))
				filecount := convert(errReader.ReadString('|'))
				seeders := convert(errReader.ReadString('|'))
				leechers := convert(errReader.ReadString('|'))
				uploaddate := convert(errReader.ReadString('|'))
				tags := errReader.ReadString('|')
				meta := errReader.ReadString('|')

				if errReader.Err != nil {
					log.Error("Failed to read post: ", errReader.Err.Error())
					break
				}

				if err != nil {
					log.Error(err.Error())
				}

				post := data.Post{
					Id:         id,
					InfoHash:   ih,
					Title:      title,
					Size:       size,
					FileCount:  filecount,
					Seeders:    seeders,
					Leechers:   leechers,
					UploadDate: uploaddate,
					Tags:       tags,
					Meta:       meta,
				}

				piece.Add(post, true)
				count++
				log.Info("post read")
			}
			ret <- &piece
		}
	}()

	return ret
}

func (c *Client) RequestAddPeer(addr string) error {
	msg := &Message{
		Header:  ProtoRequestAddPeer,
		Content: []byte(addr),
	}

	c.WriteMessage(msg)
	rep, err := c.ReadMessage()

	if err != nil {
		return err
	}

	if !rep.Ok() {
		return errors.New("Peer add request failed")
	}

	log.Info("Registered as seed peer")

	return nil
}
