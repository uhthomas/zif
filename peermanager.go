package zif

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/streamrail/concurrent-map"
	"github.com/zif/zif/data"
	"github.com/zif/zif/dht"
	"github.com/zif/zif/proto"

	log "github.com/sirupsen/logrus"
)

const HeartbeatFrequency = time.Second * 30
const AnnounceFrequency = time.Minute * 30

// errors

var (
	PeerUnreachable  = errors.New("Peer could not be reached")
	PeerDisconnected = errors.New("Peer has disconnected")
)

// handles peer connections
type PeerManager struct {
	// a map of currently connected peers
	// also use to cancel reconnects :)
	peers cmap.ConcurrentMap
	// A map of public address to Zif address
	publicToZif  cmap.ConcurrentMap
	SeedManagers cmap.ConcurrentMap

	socks     bool
	socksPort int
	localPeer *LocalPeer
}

func NewPeerManager(lp *LocalPeer) *PeerManager {
	ret := &PeerManager{}

	ret.peers = cmap.New()
	ret.publicToZif = cmap.New()
	ret.SeedManagers = cmap.New()
	ret.localPeer = lp

	return ret
}

func (pm *PeerManager) Count() int {
	return pm.peers.Count()
}

func (pm *PeerManager) Peers() map[string]*Peer {
	ret := make(map[string]*Peer)

	for k, v := range pm.peers.Items() {
		ret[k] = v.(*Peer)
	}

	return ret
}

// Given a direct address, for instance an IP or domain, connect to the peer there.
// This can be used for something like bootstrapping, or for something like
// connecting to a peer whose Zif address we have just resolved.
func (pm *PeerManager) ConnectPeerDirect(addr string) (*Peer, error) {
	var peer *Peer
	var err error

	if peer = pm.GetPeer(addr); peer != nil {
		return peer, nil
	}

	peer = &Peer{}

	if pm.socks {
		peer.streams.Socks = true
		peer.streams.SocksPort = pm.socksPort
	}

	err = peer.Connect(addr, pm.localPeer)

	if err != nil {
		return nil, PeerUnreachable
	}

	peer.ConnectClient(pm.localPeer)

	pm.SetPeer(peer)
	peer.addSeedManager = pm.AddSeedManager
	peer.addEntry = pm.localPeer.AddEntry
	peer.addSeeding = pm.localPeer.AddSeeding

	return peer, nil
}

// Resolved a Zif address into an entry, connects to the peer at the
// PublicAddress in the Entry, then return it. The peer is also stored in a map.
func (pm *PeerManager) ConnectPeer(addr dht.Address) (*Peer, *proto.Entry, error) {
	var peer *Peer

	entry, err := pm.Resolve(addr)

	if err != nil {
		return nil, nil, err
	}

	if entry.Address.Equals(pm.localPeer.Address()) {
		return nil, nil, errors.New("Cannot connect to self")
	}

	if entry == nil {
		return nil, nil, data.AddressResolutionError{entry.Address.StringOr("")}
	}

	if peer = pm.GetPeer(entry.Address.StringOr("")); peer != nil {
		return peer, entry, nil
	}

	if err != nil {
		log.Error(err.Error())
		return nil, entry, err
	}

	// now should have an entry for the peer, connect to it!
	log.WithField("address", entry.Address.StringOr("")).Debug("Connecting")

	peer, err = pm.ConnectPeerDirect(entry.PublicAddress + ":" + strconv.Itoa(entry.Port))

	// Caller can go on to choose a seed to connect to, not quite the end of the
	// world :P
	if err != nil {
		log.WithField("peer", addr.StringOr("")).Info("Failed to connect")

		return nil, entry, err
	}

	return peer, entry, nil
}

func (pm *PeerManager) GetPeer(addr string) *Peer {
	peer, has := pm.peers.Get(addr)

	if !has {
		return nil
	}

	return peer.(*Peer)
}

func (pm *PeerManager) SetPeer(p *Peer) {

	if pm.peers.Has(p.Address().StringOr("")) {
		return
	}

	pm.peers.Set(p.Address().StringOr(""), p)

	e, err := p.Entry()

	if err != nil {
		log.Error(err.Error())
		return
	}

	pm.publicToZif.Set(e.PublicAddress, p.Address().StringOr(""))

	go pm.heartbeatPeer(p)
	go pm.announcePeer(p)
}

func (pm *PeerManager) HandleCloseConnection(addr *dht.Address) {
	pm.peers.Remove(addr.StringOr(""))
}

// Pings the peer regularly to check the connection
func (pm *PeerManager) heartbeatPeer(p *Peer) {
	ticker := time.NewTicker(HeartbeatFrequency)
	defer pm.HandleCloseConnection(p.Address())

	for _ = range ticker.C {
		// just in case
		if p == nil {
			return
		}

		// If the peer has already been removed, don't bother
		if has := pm.peers.Has(p.Address().StringOr("")); !has {
			return
		}

		log.WithField("peer", p.Address().StringOr("")).Debug("Sending heartbeat")
		// allows for a suddenly slower connection, most requests have a lower timeout
		_, err := p.Ping(HeartbeatFrequency)

		if err != nil {
			log.WithField("peer", p.Address().StringOr("")).Info("Peer has no heartbeat, terminating")

			pm.HandleCloseConnection(p.Address())

			return
		}
	}
}

func (pm *PeerManager) announcePeer(p *Peer) {
	ticker := time.NewTicker(AnnounceFrequency)

	announce := func() error {
		// just in case
		if p == nil {
			return errors.New("Peer is nil")
		}

		// If the peer has already been removed, don't bother
		if has := pm.peers.Has(p.Address().StringOr("")); !has {
			return PeerDisconnected
		}

		log.WithField("peer", p.Address().StringOr("")).Info("Announcing to peer")
		err := p.Announce(pm.localPeer)

		if err != nil {
			return err
		}

		return nil
	}

	err := announce()

	if err != nil {
		log.Error(err.Error())
	}

	for _ = range ticker.C {
		err := announce()

		if err != nil {
			log.Error(err.Error())
		}
	}
}

func (pm *PeerManager) AddSeedManager(addr dht.Address) error {
	if pm.SeedManagers.Has(addr.StringOr("")) {
		return nil
	}

	log.WithField("peer", addr.StringOr("")).Info("Loading seed manager")

	sm, err := NewSeedManager(addr, pm.localPeer)

	if err != nil {
		return err
	}

	pm.SeedManagers.Set(addr.StringOr(""), pm.localPeer)
	sm.Start()

	return nil
}

func (pm *PeerManager) LoadSeeds() error {
	log.Info("Loading seed list")
	file, err := ioutil.ReadFile("./data/seeding.dat")

	if err != nil {
		return err
	}

	seedCount := len(file) / 20

	for i := 0; i < seedCount; i++ {
		addr := dht.Address{file[i*20 : 20+i*20]}

		err := pm.AddSeedManager(addr)

		if err != nil {
			log.Error(err.Error())
		}
	}
	log.Info("Finished loading seed list")

	return err
}

func (pm *PeerManager) Resolve(addr dht.Address) (*proto.Entry, error) {
	log.WithField("address", addr.StringOr("")).Debug("Resolving")

	if addr.Equals(pm.localPeer.Address()) {
		return pm.localPeer.Entry, nil
	}

	kv, err := pm.localPeer.DHT.Query(addr)

	if err != nil {
		return nil, err
	}

	if kv != nil {
		return proto.DecodeEntry(kv.Value(), false)
	}

	closest, err := pm.localPeer.DHT.FindClosest(addr)

	if err != nil {
		return nil, err
	}

	for _, i := range closest {
		e, err := proto.DecodeEntry(i.Value(), false)

		if err == nil {
			// TODO: Goroutine this.
			entry, err := pm.resolveStep(e, addr)

			if err != nil {
				log.Error(err.Error())
				continue
			}

			if entry == nil {
				continue
			}

			if entry.Address.Equals(&addr) {
				dat, err := entry.Encode()

				if err != nil {
					return nil, err
				}

				kv := dht.NewKeyValue(entry.Address, dat)

				pm.localPeer.DHT.Insert(kv)

				return entry, nil
			}
		}
	}

	return nil, errors.New("Address could not be resolved")
}

// Will return the entry itself, or an error.
func (pm *PeerManager) resolveStep(e *proto.Entry, addr dht.Address) (*proto.Entry, error) {
	// connect to the peer
	var peer *Peer
	var err error

	peer = pm.GetPeer(e.Address.StringOr(""))

	if peer == nil {
		peer, err = pm.ConnectPeerDirect(fmt.Sprintf("%s:%d", e.PublicAddress, e.Port))

		if err != nil {
			return nil, err
		}
	}

	kv, err := peer.Query(addr)

	if err != nil {
		return nil, err
	}

	if kv != nil {
		entry := kv
		return entry.(*proto.Entry), err
	}

	closest, err := peer.FindClosest(addr)

	if err != nil {
		return nil, err
	}

	for _, i := range closest {
		entry := i.(*proto.Entry)

		result, err := pm.resolveStep(entry, addr)

		if err != nil {
			return nil, err
		}

		if result != nil {
			return result, nil
		}
	}

	return nil, errors.New("No entries could be found")
}
