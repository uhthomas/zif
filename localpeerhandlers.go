package zif

import (
	"bufio"
	"compress/gzip"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"

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

	address := dht.Address{}
	err := msg.Read(&address)

	if err != nil {
		return err
	}

	log.WithField("target", address.StringOr("")).Info("Recieved query")

	ok := &proto.Message{Header: proto.ProtoOk}
	err = cl.WriteMessage(ok)

	if err != nil {
		return err
	}

	if address.Equals(lp.Address()) {
		log.WithField("name", lp.Entry.Name).Debug("Query for local peer")

		msg := &proto.Message{Header: proto.ProtoDhtQuery}

		err = msg.Write(lp.Entry)

		if err != nil {
			return err
		}

		err = cl.WriteMessage(msg)

	} else {
		kv, err := lp.DHT.Query(address)

		if err != nil {
			return err
		}

		if kv == nil {
			cl.WriteMessage(&proto.Message{Header: proto.ProtoNo})
		}

		msg := &proto.Message{Header: proto.ProtoDhtQuery}

		err = msg.Write(kv)

		if err != nil {
			return err
		}

		err = cl.WriteMessage(msg)
	}

	return err
}

func (lp *LocalPeer) HandleFindClosest(msg *proto.Message) error {

	cl := msg.Client

	address := dht.Address{}
	err := msg.Read(&address)

	if err != nil {
		return err
	}

	log.WithField("target", address.StringOr("")).Info("Recieved find closest")

	ok := &proto.Message{Header: proto.ProtoOk}
	err = cl.WriteMessage(ok)

	if err != nil {
		return err
	}

	log.Debug("Accepted address")

	results := &proto.Message{
		Header: proto.ProtoEntry,
	}

	if address.Equals(lp.Address()) {
		log.WithField("name", lp.Entry.Name).Debug("Query for local peer")

		dat, err := lp.Entry.Encode()

		if err != nil {
			return err
		}

		results.Write(1)

		err = cl.WriteMessage(results)

		if err != nil {
			return err
		}

		err = cl.WriteMessage(dat)

	} else {
		log.WithField("peer", address.StringOr("")).Debug("FindClosest for")
		pairs, err := lp.DHT.FindClosest(address)

		if err != nil {
			return err
		}

		log.WithField("count", len(pairs)).Debug("Found entries")

		results.Write(len(pairs))

		err = cl.WriteMessage(results)

		if err != nil {
			return err
		}

		log.Debug("Wrote length to peer")

		for _, kv := range pairs {
			err = cl.WriteMessage(kv)
		}
	}

	return err
}

func (lp *LocalPeer) HandleAnnounce(msg *proto.Message) error {
	cl, err := proto.NewClient(msg.Stream)

	if err != nil {
		return err
	}

	defer msg.Stream.Close()

	entry := dht.Entry{}
	err = msg.Read(&entry)

	log.WithField("address", entry.Address.StringOr("")).Info("Announce")

	if err != nil {
		return err
	}

	affected, err := lp.DHT.Insert(entry)

	if err == nil && affected > 0 {
		cl.WriteMessage(&proto.Message{Header: proto.ProtoOk})
		log.WithField("peer", entry.Address.StringOr("")).Info("Saved new peer")

	} else {
		cl.WriteMessage(&proto.Message{Header: proto.ProtoNo})
		return errors.New("Failed to save entry")
	}

	return nil

}

func (lp *LocalPeer) HandleSearch(msg *proto.Message) error {

	sq := proto.MessageSearchQuery{}
	err := msg.Read(&sq)

	if err != nil {
		return err
	}

	log.WithField("query", sq.Query).Info("Search recieved")

	posts, err := lp.Database.Search(sq.Query, sq.Page, 25)

	if err != nil {
		return err
	}
	log.Info("Posts loaded")

	post_msg := &proto.Message{
		Header: proto.ProtoPosts,
	}

	err = post_msg.Write(posts)

	if err != nil {
		return err
	}

	return msg.Client.WriteMessage(post_msg)
}

func (lp *LocalPeer) HandleRecent(msg *proto.Message) error {
	log.Info("Recieved query for recent posts")

	page := 0
	err := msg.Read(&page)

	if err != nil {
		return err
	}

	recent, err := lp.Database.QueryRecent(page)

	if err != nil {
		return err
	}

	resp := &proto.Message{
		Header: proto.ProtoPosts,
	}

	err = resp.Write(recent)

	if err != nil {
		return err
	}

	return msg.Client.WriteMessage(resp)
}

func (lp *LocalPeer) HandlePopular(msg *proto.Message) error {
	log.Info("Recieved query for popular posts")

	page := 0
	err := msg.Read(&page)

	if err != nil {
		return err
	}

	recent, err := lp.Database.QueryPopular(page)

	if err != nil {
		return err
	}

	resp := &proto.Message{
		Header: proto.ProtoPosts,
	}

	err = resp.Write(recent)

	if err != nil {
		return err
	}

	return msg.Client.WriteMessage(resp)
}

func (lp *LocalPeer) HandleHashList(msg *proto.Message) error {
	address := dht.Address{}
	err := msg.Read(&address)

	if err != nil {
		return err
	}

	log.WithField("address", address.StringOr("")).Info("Collection request recieved")

	var sig []byte
	var hash []byte
	var hashList []byte

	entry, err := lp.DHT.Query(address)

	// could be the local peer
	// if it is not, that is dealt with below :)
	if err == sql.ErrNoRows {
		err = nil
	}

	if err != nil {
		return err
	}

	if address.Equals(lp.Address()) {
		log.Info("Collection request for local peer")
		sig = lp.Entry.CollectionSig
		hash = lp.Collection.Hash()
		hashList = lp.Collection.HashList

	} else if entry != nil {
		// load the hashlist from disk, if it exists. If not, err
		// if not "err", then it'd probably read its own collection
		hl, err := ioutil.ReadFile(fmt.Sprintf("./data/%s/collection.dat", address.StringOr("err")))

		if err != nil {
			return err
		}

		hashList = make([]byte, len(hl))
		copy(hashList, hl)

		sig = make([]byte, len(entry.CollectionSig))
		copy(sig, entry.CollectionSig)

		hash = make([]byte, len(entry.CollectionHash))
		copy(hash, entry.CollectionHash)

	} else {
		return errors.New("Cannot return collection hash list")
	}

	mhl := proto.MessageCollection{
		Hash:      hash,
		HashList:  hashList,
		Size:      len(hashList) / 32,
		Signature: sig,
	}

	resp := &proto.Message{
		Header: proto.ProtoHashList,
	}

	resp.Write(mhl)

	if err != nil {
		return err
	}

	msg.Client.WriteMessage(resp)

	return nil
}

func (lp *LocalPeer) HandlePiece(msg *proto.Message) error {

	mrp := proto.MessageRequestPiece{}
	err := msg.Read(&mrp)

	log.WithFields(log.Fields{
		"id":     mrp.Id,
		"length": mrp.Length,
	}).Info("Recieved piece request")

	if err != nil {
		return err
	}

	var posts chan *data.Post

	if mrp.Address == lp.Address().StringOr("") {
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

	address := dht.Address{}
	err := msg.Read(&address)

	if err != nil {
		return err
	}

	from, _ := msg.From.String()
	pfor, _ := address.String()
	log.WithFields(log.Fields{"from": from, "for": pfor}).Info("Handling add peer request")

	if address.Equals(lp.Address()) {

		add := true

		for _, i := range lp.Entry.Seeds {
			if msg.From.Equals(&dht.Address{Raw: i}) {
				add = false
			}
		}

		if add {
			b, _ := msg.From.Bytes()
			lp.Entry.Seeds = append(lp.Entry.Seeds, b)
		}

		err := lp.SaveEntry()
		if err != nil {
			return err
		}

		log.WithField("peer", from).Info("New seed peer")

	} else {
		// then we need to see if we have the entry for that address
		entry, err := lp.DHT.Query(address)

		if err != nil {
			return err
		}

		if entry == nil {
			return errors.New("Cannot add peer, do not have entry")
		}

		// read the address of the peer as raw bytes, and add it to the seed list
		b, _ := msg.From.Bytes()
		entry.Seeds = append(entry.Seeds, b)

		log.WithFields(
			log.Fields{
				"for":  pfor,
				"seed": from}).Info("Added seed")

		lp.DHT.Insert(*entry)
	}

	msg.Client.WriteMessage(&proto.Message{Header: proto.ProtoOk})

	return nil
}

func (lp *LocalPeer) HandleHandshake(header proto.ConnHeader) (proto.NetworkPeer, error) {
	peer := &Peer{}
	peer.SetTCP(header)
	_, err := peer.ConnectServer()

	if err != nil {
		return nil, err
	}

	lp.peerManager.SetPeer(peer)

	// we have a "free" entry, insert it! Just in case :D
	entry, err := peer.Entry()
	if err != nil {
		return nil, err
	}

	lp.DHT.Insert(*entry)

	return peer, nil
}

func (lp *LocalPeer) ListenStream(peer *Peer) {
	lp.Server.ListenStream(peer, lp)
}
