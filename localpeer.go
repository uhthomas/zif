// The local peer. This runs on the current node, so we have access to its
// private key, database, etc.

package libzif

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"regexp"

	log "github.com/sirupsen/logrus"
	"github.com/streamrail/concurrent-map"
	"golang.org/x/crypto/ed25519"

	"github.com/zif/zif/data"
	"github.com/zif/zif/dht"
	"github.com/zif/zif/jobs"
	"github.com/zif/zif/proto"
)

const ResolveListSize = 1

type LocalPeer struct {
	Peer
	Entry         *proto.Entry
	DHT           *dht.DHT
	Server        proto.Server
	Collection    *data.Collection
	Database      *data.Database
	PublicAddress string
	// These are the databases of all of the peers that we have mirrored.
	Databases   cmap.ConcurrentMap
	Collections cmap.ConcurrentMap

	SearchProvider *data.SearchProvider

	privateKey  ed25519.PrivateKey
	peerManager *PeerManager
}

func (lp *LocalPeer) Setup() {
	var err error

	lp.Entry = &proto.Entry{}
	lp.Entry.Signature = make([]byte, ed25519.SignatureSize)

	lp.Databases = cmap.New()
	lp.Collections = cmap.New()

	lp.peerManager = NewPeerManager(lp)

	lp.Address().Generate(lp.PublicKey())

	lp.DHT = dht.NewDHT(lp.address, "./data/dht")
	lp.DHT.LoadTable("./data/dht/table.dat")

	if err != nil {
		panic(err)
	}

	lp.Collection, err = data.LoadCollection("./data/collection.dat")

	if err != nil {
		lp.Collection = data.NewCollection()
		log.Info("Created new collection")
	}

	// Loop through all the databases of other peers in ./data, load them.
	handler := func(path string, info os.FileInfo, err error) error {
		if path != "data/posts.db" && info.Name() == "posts.db" {
			r, err := regexp.Compile("data/(\\w+)/.+")

			if err != nil {
				return err
			}

			addr := r.FindStringSubmatch(path)

			db := data.NewDatabase(path)

			err = db.Connect()

			if err != nil {
				return err
			}

			if len(addr) < 2 {
				return nil
			}

			lp.Databases.Set(addr[1], db)
		}
		return nil
	}

	filepath.Walk("./data", handler)

	// TODO: This does not work without internet xD
	/*if lp.Entry.PublicAddress == "" {
		log.Debug("Local peer public address is nil, attempting to fetch")
		ip := external_ip()
		log.Debug("External IP is ", ip)
		lp.Entry.PublicAddress = ip
	}*/

	lp.SearchProvider = data.NewSearchProvider()
}

func (lp *LocalPeer) SignEntry() {
	data, _ := lp.Entry.Bytes()
	copy(lp.Entry.Signature, ed25519.Sign(lp.privateKey, data))
}

// Sign any bytes.
func (lp *LocalPeer) Sign(msg []byte) []byte {
	return ed25519.Sign(lp.privateKey, msg)
}

// Pass the address to listen on. This is for the Zif connection.
func (lp *LocalPeer) Listen(addr string) {
	lp.SignEntry()
	go lp.Server.Listen(addr, lp, lp.Entry)
}

// Generate a ed25519 keypair.
func (lp *LocalPeer) GenerateKey() {
	var err error

	lp.publicKey, lp.privateKey, err = ed25519.GenerateKey(nil)

	if err != nil {
		panic(err)
	}
}

// Writes the private key to a file, in this way persisting your identity -
// all the other addresses can be generated from this, no need to save them.
// By default this file is "identity.dat"
func (lp *LocalPeer) WriteKey() error {
	if len(lp.privateKey) == 0 {
		return errors.
			New("LocalPeer does not have a private key, please generate")
	}

	err := ioutil.WriteFile("identity.dat", lp.privateKey, 0400)

	return err
}

// Read the private key from file. This is the "identity.dat" file. The public
// key is also then generated from the private key.
func (lp *LocalPeer) ReadKey() error {
	pk, err := ioutil.ReadFile("identity.dat")

	if err != nil {
		return err
	}

	lp.privateKey = pk
	lp.publicKey = lp.privateKey.Public().(ed25519.PublicKey)

	return nil
}

// At the moment just query for the closest known peer
// This takes a Zif address as a string and attempts to resolve it to an entry.
// This may be fast, may be a little slower. Will recurse its way through as
// many Queries as needed, getting closer to the target until either it cannot
// be found or is found.
// Cannot be found if a Query returns nothing, in this case the address does not
// exist on the DHT. Otherwise we should get to a peer that either has the entry,
// or one that IS the peer we are hunting.
// Takes a string as the API will just be passing a Zif address as a string.
// May well change, I'm unsure really. Pretty happy with it at the moment though.
// TODO: Somehow move this to the DHT package.
func (lp *LocalPeer) Resolve(addr string) (*proto.Entry, error) {
	log.WithField("address", addr).Debug("Resolving")

	lps, _ := lp.Address().String()
	if addr == lps {
		return lp.Entry, nil
	}

	address := dht.DecodeAddress(addr)

	kv, err := lp.DHT.Query(address)

	if err != nil {
		return nil, err
	}

	if kv != nil {
		return proto.JsonToEntry(kv.Value())
	}

	closest, err := lp.DHT.FindClosest(address)

	if err != nil {
		return nil, err
	}

	for _, i := range closest {
		e, err := proto.JsonToEntry(i.Value())

		if err == nil {
			// TODO: Goroutine this.
			entry, err := lp.resolveStep(e, address)

			if err != nil {
				log.Error(err.Error())
				continue
			}

			if entry.Address.Equals(&address) {
				return entry, nil
			}
		}
	}

	return nil, errors.New("Address could not be resolved")
}

// Will return the entry itself, or an error.
func (lp *LocalPeer) resolveStep(e *proto.Entry, addr dht.Address) (*proto.Entry, error) {
	// connect to the peer
	var peer *Peer
	var err error

	es, _ := e.Address.String()
	peer = lp.GetPeer(es)

	if peer == nil {
		peer, err = lp.ConnectPeerDirect(fmt.Sprintf("%s:%d", e.PublicAddress, e.Port))

		if err != nil {
			return nil, err
		}
	}

	s, _ := addr.String()
	kv, err := peer.Query(s)

	if err != nil {
		return nil, err
	}

	if kv != nil {
		entry, err := proto.JsonToEntry(kv.Value())
		return entry, err
	}

	closest, err := peer.FindClosest(s)

	if err != nil {
		return nil, err
	}

	for _, i := range closest {
		entry, err := proto.JsonToEntry(i.Value())

		if err != nil {
			continue
		}

		result, err := lp.resolveStep(entry, addr)

		if result != nil {
			ret, _ := proto.JsonToEntry(i.Value())

			return ret, nil
		}
	}

	return nil, errors.New("No entries could be found")
}

func (lp *LocalPeer) SaveEntry() error {
	dat, err := lp.Entry.Json()

	if err != nil {
		return err
	}

	return ioutil.WriteFile("./data/entry.json", dat, 0644)
}

func (lp *LocalPeer) LoadEntry() error {
	dat, err := ioutil.ReadFile("./data/entry.json")

	if err != nil {
		return err
	}

	entry, err := proto.JsonToEntry(dat)

	if err != nil {
		return err
	}

	lp.Entry = entry

	return nil
}

func (lp *LocalPeer) Close() {
	lp.CloseStreams()
	lp.DHT.SaveTable("./data/dht/table.dat")
	lp.Server.Close()
	lp.Database.Close()
}

func (lp *LocalPeer) AddPost(p data.Post, store bool) (int64, error) {
	log.WithField("Title", p.Title).Info("Adding post")

	valid := p.Valid()

	if valid != nil {
		return -1, valid
	}

	lp.Entry.PostCount += 1

	id, err := lp.Database.InsertPost(p)

	pieceIndex := int(math.Floor(float64(id) / float64(data.PieceSize)))
	piece, err := lp.Database.QueryPiece(uint(pieceIndex), false)

	lp.Collection.Add(piece)
	lp.Collection.Rehash()
	lp.Collection.Save("./data/collection.dat")

	if err != nil {
		return id, err
	}

	lp.SignEntry()
	err = lp.SaveEntry()

	return id, err
}

func (lp *LocalPeer) StartExploring() {
	in := make(chan dht.KeyValue, jobs.ExploreBufferSize)

	lp.seedExplore(in)

	ret := jobs.ExploreJob(in,
		func(addr string) (interface{}, error) { return lp.ConnectPeer(addr) },
		lp.address,
		func(in chan dht.KeyValue) { lp.seedExplore(in) })

	go func() {
		for i := range ret {
			has := lp.DHT.Has(*i.Key())

			if i.Key().Equals(lp.Address()) {
				continue
			}

			// reinsert regardless of whether we have it or not. This helps
			// keep more "active" things at the top, and also keeps us up to date.
			lp.DHT.Insert(&i)

			if !has {
				ps, _ := i.Key().String()
				log.WithField("peer", ps).Info("Discovered new peer")
				in <- i
			}
		}
	}()
}

func (lp *LocalPeer) seedExplore(in chan dht.KeyValue) {
	closest, err := lp.DHT.FindClosest(*lp.Address())

	if err != nil {
		panic(err)
	}

	addr, _ := dht.RandomAddress()
	closestRand, err := lp.DHT.FindClosest(*addr)

	closest = append(closest, closestRand...)

	for _, i := range closest {
		if !i.Key().Equals(lp.Address()) {
			in <- *i
		}
	}
}

// convenience methods
func (lp *LocalPeer) PeerCount() int {
	return lp.peerManager.Count()
}

func (lp *LocalPeer) Peers() map[string]*Peer {
	return lp.peerManager.Peers()
}

func (lp *LocalPeer) ConnectPeerDirect(addr string) (*Peer, error) {
	return lp.peerManager.ConnectPeerDirect(addr)
}

func (lp *LocalPeer) HandleCloseConnection(addr *dht.Address) {
	lp.peerManager.HandleCloseConnection(addr)
}

func (lp *LocalPeer) SetPeer(p *Peer) {
	lp.peerManager.SetPeer(p)
}

func (lp *LocalPeer) SetNetworkPeer(p proto.NetworkPeer) {
	switch p.(type) {
	case *Peer:
		lp.peerManager.SetPeer(p.(*Peer))
	default:
		log.Error("NetworkPeer is not *Peer")
	}
}

func (lp *LocalPeer) GetPeer(addr string) *Peer {
	return lp.peerManager.GetPeer(addr)
}

func (lp *LocalPeer) GetNetworkPeer(addr string) proto.NetworkPeer {
	return lp.peerManager.GetPeer(addr)
}

func (lp *LocalPeer) SetSocks(on bool) {
	lp.peerManager.socks = on
}

func (lp *LocalPeer) SetSocksPort(port int) {
	lp.peerManager.socksPort = port
}

func (lp *LocalPeer) GetSocksPort() int {
	return lp.peerManager.socksPort
}
