package libzif

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/zif/zif/data"
	"github.com/zif/zif/dht"
	"github.com/zif/zif/proto"
)

const MaxSearchLength = 256

// TODO: Move this into some sort of handler object, can handle general requests.

// TODO: While I think about it, move all these TODOs to issues or a separate
// file/issue tracker or something.

// Querying peer sends a Zif address
// This peer will respond with a list of the k closest peers, ordered by distance.
// The top peer may well be the one that is being queried for :)
func (lp *LocalPeer) HandleQuery(msg *proto.Message) error {
	log.Info("Handling query")
	cl := msg.Client

	//msg.From.limiter.queryLimiter.Wait()

	address := dht.DecodeAddress(string(msg.Content))
	s, _ := address.String()
	log.WithField("target", s).Info("Recieved query")

	ok := &proto.Message{Header: proto.ProtoOk}
	err := cl.WriteMessage(ok)

	if err != nil {
		return err
	}

	if address.Equals(lp.Address()) {
		log.WithField("name", lp.Entry.Name).Debug("Query for local peer")

		var dat []byte
		dat, err = lp.Entry.Json()

		if err != nil {
			return err
		}

		kv := dht.NewKeyValue(lp.Entry.Address, dat)
		encoded, _ := json.Marshal(kv)
		err = cl.WriteMessage(&proto.Message{Header: proto.ProtoDhtQuery, Content: encoded})

	} else {
		kv := &dht.KeyValue{}
		kv, err = lp.DHT.Query(address)

		if err != nil {
			return err
		}

		if kv == nil {
			cl.WriteMessage(&proto.Message{Header: proto.ProtoNo})
		}

		encoded, _ := json.Marshal(kv)
		err = cl.WriteMessage(&proto.Message{Header: proto.ProtoDhtQuery, Content: encoded})
	}

	return err
}

func (lp *LocalPeer) HandleFindClosest(msg *proto.Message) error {
	cl := msg.Client

	address := dht.DecodeAddress(string(msg.Content))
	s, _ := address.String()
	log.WithField("target", s).Info("Recieved find closest")

	ok := &proto.Message{Header: proto.ProtoOk}
	err := cl.WriteMessage(ok)

	if err != nil {
		return err
	}

	log.Debug("Accepted address")

	results := &proto.Message{
		Header: proto.ProtoEntry,
	}

	if address.Equals(lp.Address()) {
		log.WithField("name", lp.Entry.Name).Debug("Query for local peer")

		json, err := lp.Entry.Json()

		if err != nil {
			return err
		}

		results.WriteInt(1)

		err = cl.WriteMessage(results)

		if err != nil {
			return err
		}

		kv := dht.NewKeyValue(lp.Entry.Address, json)

		err = cl.WriteMessage(kv)

	} else {
		pairs, err := lp.DHT.FindClosest(address)

		if err != nil {
			return err
		}

		results.WriteInt(len(pairs))

		err = cl.WriteMessage(results)

		if err != nil {
			return err
		}

		for _, kv := range pairs {
			err = cl.WriteMessage(kv)
		}
	}

	return err
}

func (lp *LocalPeer) HandleAnnounce(msg *proto.Message) error {
	cl := proto.NewClient(msg.Stream)

	defer msg.Stream.Close()

	entry := Entry{}
	err := msg.Decode(&entry)

	es, _ := entry.Address.String()
	log.WithField("address", es).Info("Announce")

	if err != nil {
		return err
	}

	json, _ := entry.Json()
	err = lp.DHT.Insert(dht.NewKeyValue(entry.Address, json))

	if err == nil {
		cl.WriteMessage(&proto.Message{Header: proto.ProtoOk})
		log.WithField("peer", es).Info("Saved new peer")

	} else {
		cl.WriteMessage(&proto.Message{Header: proto.ProtoNo})
		return errors.New("Failed to save entry")
	}

	return nil

}

func (lp *LocalPeer) HandleSearch(msg *proto.Message) error {
	if len(msg.Content) > MaxSearchLength {
		return errors.New("Search query too long")
	}

	sq := proto.MessageSearchQuery{}
	err := msg.Decode(&sq)

	if err != nil {
		return err
	}

	log.WithField("query", sq.Query).Info("Search recieved")

	posts, err := lp.Database.Search(sq.Query, sq.Page, 25)

	if err != nil {
		return err
	}
	log.Info("Posts loaded")

	json, err := json.Marshal(posts)

	if err != nil {
		return err
	}

	post_msg := &proto.Message{
		Header:  proto.ProtoPosts,
		Content: json,
	}

	msg.Client.WriteMessage(post_msg)

	return nil
}

func (lp *LocalPeer) HandleRecent(msg *proto.Message) error {
	log.Info("Recieved query for recent posts")

	page, err := strconv.Atoi(string(msg.Content))

	if err != nil {
		return err
	}

	recent, err := lp.Database.QueryRecent(page)

	if err != nil {
		return err
	}

	recent_json, err := json.Marshal(recent)

	if err != nil {
		return err
	}

	resp := &proto.Message{
		Header:  proto.ProtoPosts,
		Content: recent_json,
	}

	msg.Client.WriteMessage(resp)

	return nil
}

func (lp *LocalPeer) HandlePopular(msg *proto.Message) error {
	log.Info("Recieved query for popular posts")

	page, err := strconv.Atoi(string(msg.Content))

	if err != nil {
		return err
	}

	recent, err := lp.Database.QueryPopular(page)

	if err != nil {
		return err
	}

	recent_json, err := json.Marshal(recent)

	if err != nil {
		return err
	}

	resp := &proto.Message{
		Header:  proto.ProtoPosts,
		Content: recent_json,
	}

	msg.Client.WriteMessage(resp)

	return nil
}

func (lp *LocalPeer) HandleHashList(msg *proto.Message) error {
	address := dht.Address{msg.Content}

	s, _ := address.String()
	log.WithField("address", s).Info("Collection request recieved")

	var sig []byte

	if address.Equals(lp.Address()) {
		sig = lp.Sign(lp.Collection.HashList)
	} else {
		// this means that the hash list wanted does not belong to this peer
		// TODO: sort out getting a hash list for a peer that has been mirrored
	}

	mhl := proto.MessageCollection{lp.Collection.Hash(), lp.Collection.HashList, len(lp.Collection.HashList) / 32, sig}
	data, err := mhl.Encode()

	if err != nil {
		return err
	}

	resp := &proto.Message{
		Header:  proto.ProtoHashList,
		Content: data,
	}

	msg.Client.WriteMessage(resp)

	return nil
}

func (lp *LocalPeer) HandlePiece(msg *proto.Message) error {

	mrp := proto.MessageRequestPiece{}
	err := msg.Decode(&mrp)

	log.WithFields(log.Fields{
		"id":     mrp.Id,
		"length": mrp.Length,
	}).Info("Recieved piece request")

	if err != nil {
		return err
	}

	var posts chan *data.Post

	lps, _ := lp.Entry.Address.String()
	if mrp.Address == lps {
		posts = lp.Database.QueryPiecePosts(mrp.Id, mrp.Length, true)

	} else if lp.Databases.Has(mrp.Address) {
		db, _ := lp.Databases.Get(mrp.Address)
		posts = db.(*data.Database).QueryPiecePosts(mrp.Id, mrp.Length, true)

	} else {
		return errors.New("Piece not found")
	}

	// Buffered writer -> gzip -> net
	// or
	// gzip -> buffered writer -> net
	// I'm guessing the latter allows for gzip to maybe run a little faster?
	// The former may allow for database reads to occur a little faster though.
	// buffer both?
	bw := bufio.NewWriter(msg.Stream)
	gzw := gzip.NewWriter(bw)

	for i := range posts {
		i.Write("|", "", gzw)
	}

	(&data.Post{Id: -1}).Write("|", "", gzw)

	gzw.Flush()
	bw.Flush()

	log.Info("Sent all")

	return nil
}

func (lp *LocalPeer) HandleAddPeer(msg *proto.Message) error {
	// The AddPeer message contains the address of the peer that the client
	// wishes to be registered for.

	peerFor := string(msg.Content)

	log.Info("Handling add peer request for ", peerFor)

	// First up, we need the address in binary form
	address := dht.DecodeAddress(peerFor)
	s, _ := address.String()

	if len(address.Raw) != dht.AddressBinarySize {
		msg.Client.WriteMessage(&proto.Message{Header: proto.ProtoNo})
		return errors.New("Invalid binary address size")
	}

	if address.Equals(lp.Address()) {
		log.WithField("peer", s).Info("New seed peer")

		add := true

		for _, i := range lp.Entry.Seeds {
			if address.Equals(&dht.Address{i}) {
				add = false
			}
		}

		if add {
			b, _ := address.Bytes()
			lp.Entry.Seeds = append(lp.Entry.Seeds, b)
		}

	} else {
		// then we need to see if we have the entry for that address
		kv, err := lp.DHT.Query(address)

		if err != nil {
			return err
		}

		decoded, err := JsonToEntry(kv.Value())

		if err != nil {
			return err
		}

		// if the routing table contains the address we are looking for,
		// register a new seed.
		if decoded.Address.Equals(&address) {
			b, _ := address.Bytes()
			decoded.Seeds = append(decoded.Seeds, b)
		}

		json, err := decoded.Json()
		if err != nil {
			return err
		}

		lp.DHT.Insert(dht.NewKeyValue(address, json))
	}

	msg.Client.WriteMessage(&proto.Message{Header: proto.ProtoOk})

	return nil
}

func (lp *LocalPeer) HandlePing(msg *proto.Message) error {
	s, _ := msg.From.String()
	log.WithField("peer", s).Info("Ping")

	return msg.Client.WriteMessage(&proto.Message{Header: proto.ProtoPong})
}

func (lp *LocalPeer) HandleCloseConnection(addr *dht.Address) {
	s, _ := addr.String()
	lp.Peers.Remove(s)
}

func (lp *LocalPeer) HandleHandshake(header proto.ConnHeader) (proto.NetworkPeer, error) {
	peer := &Peer{}
	peer.SetTCP(header)
	_, err := peer.ConnectServer()

	if err != nil {
		return nil, err
	}

	s, _ := peer.Address().String()
	lp.Peers.Set(s, peer)

	return peer, nil
}

func (lp *LocalPeer) ListenStream(peer *Peer) {
	lp.Server.ListenStream(peer, lp)
}
