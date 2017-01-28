// The local peer. This runs on the current node, so we have access to its
// private key, database, etc.

package zif

import (
	"errors"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/streamrail/concurrent-map"
	"golang.org/x/crypto/ed25519"

	"github.com/zif/zif/data"
	"github.com/zif/zif/dht"
	"github.com/zif/zif/jobs"
	"github.com/zif/zif/proto"
	"github.com/zif/zif/util"
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
	seedManager *SeedManager
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

		} else if path != "data/collection.dat" && info.Name() == "collection.dat" {
			r, err := regexp.Compile("data/(\\w+)/.+")

			if err != nil {
				return err
			}

			addr := r.FindStringSubmatch(path)

			dat, err := ioutil.ReadFile(path)

			if err != nil {
				return err
			}

			lp.Collections.Set(addr[1], dat)
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
	lp.Entry.Updated = uint64(time.Now().Unix())
	data, _ := lp.Entry.Bytes()
	copy(lp.Entry.Signature, ed25519.Sign(lp.privateKey, data))
}

// Sign any bytes.
func (lp *LocalPeer) Sign(msg []byte) []byte {
	return ed25519.Sign(lp.privateKey, msg)
}

// Pass the address to listen on. This is for the Zif connection.
func (lp *LocalPeer) Listen(addr string) {
	var err error
	lp.seedManager, err = NewSeedManager(lp.Entry.Address, lp)

	if err != nil {
		panic(err)
	}

	lp.SignEntry()
	go lp.Server.Listen(addr, lp, lp.Entry)
	go lp.QuerySelf()
	go lp.peerManager.LoadSeeds()

	lp.seedManager.Start()
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

func (lp *LocalPeer) SaveEntry() error {
	dat, err := lp.Entry.EncodeString()

	if err != nil {
		return err
	}

	return ioutil.WriteFile("./data/entry.json", []byte(dat), 0644)
}

func (lp *LocalPeer) LoadEntry() error {
	dat, err := ioutil.ReadFile("./data/entry.json")

	if err != nil {
		return err
	}

	entry, err := proto.DecodeEntry(dat, true)

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

	hash := lp.Collection.Hash()
	sig := lp.Sign(lp.Collection.Hash())

	lp.Entry.CollectionSig = make([]byte, len(sig))
	copy(lp.Entry.CollectionSig, sig)

	lp.Entry.CollectionHash = make([]byte, len(hash))
	copy(lp.Entry.CollectionHash, hash)

	if err != nil {
		return id, err
	}

	lp.SignEntry()
	err = lp.SaveEntry()

	return id, err
}

func (lp *LocalPeer) StartExploring() {
	in := make(chan proto.Entry, jobs.ExploreBufferSize)

	lp.seedExplore(in)

	ret := jobs.ExploreJob(in,
		func(addr dht.Address) (interface{}, error) {
			peer, _, err := lp.ConnectPeer(addr)
			return peer, err
		},
		lp.address,
		func(in chan proto.Entry) { lp.seedExplore(in) })

	go func() {
		for i := range ret {
			kv, err := lp.DHT.Query(i.Address)

			if i.Address.Equals(lp.Address()) {
				continue
			}

			ps, _ := i.Address.String()
			encoded, err := i.Encode()

			if err != nil {
				log.Error(err.Error())
				continue
			}

			if kv == nil {

				if err != nil {
					log.Error(err.Error())
					continue
				}

				lp.DHT.Insert(dht.NewKeyValue(i.Address, encoded))
				log.WithField("peer", ps).Info("Discovered new peer")

				in <- i
			} else {
				current, err := proto.DecodeEntry(kv.Value(), false)
				if err != nil {
					continue
				}

				if i.Updated > current.Updated {
					lp.DHT.Insert(dht.NewKeyValue(i.Address, encoded))
					log.WithField("peer", ps).Info("Updated peer")

				} else if len(i.Seeds) > len(current.Seeds) {
					current.Seeds = util.MergeSeeds(current.Seeds, i.Seeds)

					currentDat, err := current.Encode()

					if err != nil {
						log.Error(err.Error())
						continue
					}

					lp.DHT.Insert(dht.NewKeyValue(current.Address, currentDat))

					log.WithField("peer", ps).Info("Found new seeds")
				}
			}
		}
	}()
}

func (lp *LocalPeer) seedExplore(in chan proto.Entry) {
	closest, err := lp.DHT.FindClosest(*lp.Address())

	if err != nil {
		panic(err)
	}

	addr, _ := dht.RandomAddress()
	closestRand, err := lp.DHT.FindClosest(*addr)

	closest = append(closest, closestRand...)
	log.WithField("seeds", len(closest)).Info("Seeding peer explore")

	for _, i := range closest {
		if i == nil {
			continue
		}

		if !i.Key().Equals(lp.Address()) {
			entry := proto.Entry{}
			err := i.Decode(&entry)

			if err != nil {
				log.Error(err.Error())
				continue
			}

			in <- entry
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

func (lp *LocalPeer) ConnectPeer(addr dht.Address) (*Peer, *proto.Entry, error) {
	return lp.peerManager.ConnectPeer(addr)
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

func (lp *LocalPeer) Resolve(addr dht.Address) (*proto.Entry, error) {
	return lp.peerManager.Resolve(addr)
}

func (lp *LocalPeer) QueryEntry(addr dht.Address) (*proto.Entry, error) {
	if addr.Equals(lp.Address()) {
		return lp.Entry, nil
	}

	kv, err := lp.DHT.Query(addr)

	if err != nil {
		return nil, err
	}

	if kv == nil {
		return nil, errors.New("No entry with address")
	}

	return proto.DecodeEntry(kv.Value(), false)
}

func (lp *LocalPeer) QuerySelf() {
	log.Info("Querying for seeds")
	ticker := time.NewTicker(time.Minute * 5)

	for _ = range ticker.C {
		if len(lp.Entry.Seeds) == 0 {
			continue
		}

		i := lp.Entry.Seeds[util.CryptoRandInt(0, int64(len(lp.Entry.Seeds)))]

		addr := dht.Address{i}

		if addr.Equals(lp.Address()) {
			continue
		}

		s, err := addr.String()

		if err != nil {
			continue
		}

		log.WithField("peer", s).Info("Querying for new feeds for self")

		peer, _, err := lp.ConnectPeer(addr)

		if err != nil {
			continue
		}

		e, err := peer.Query(*lp.Address())

		if err != nil {
			continue
		}
		entry := e.(*proto.Entry)

		if len(entry.Seeds) > len(lp.Entry.Seeds) {
			log.WithField("from", s).Info("Found new seeds for self")
			lp.Entry.Seeds = util.MergeSeeds(lp.Entry.Seeds, entry.Seeds)
		}

		time.Sleep(time.Minute * 5)
	}
}

func (lp *LocalPeer) AddEntry(entry *proto.Entry) error {
	dat, err := entry.Encode()

	if err != nil {
		return err
	}

	kv := dht.NewKeyValue(entry.Address, dat)

	return lp.DHT.Insert(kv)
}
