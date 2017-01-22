package libzif

import (
	"strconv"

	"github.com/streamrail/concurrent-map"
	"github.com/zif/zif/data"
	"github.com/zif/zif/dht"

	log "github.com/sirupsen/logrus"
)

// handles peer connections
type PeerManager struct {
	// a map of currently connected peers
	// also use to cancel reconnects :)
	peers cmap.ConcurrentMap
	// A map of public address to Zif address
	publicToZif cmap.ConcurrentMap

	socks     bool
	socksPort int
	localPeer *LocalPeer
}

func NewPeerManager(lp *LocalPeer) *PeerManager {
	ret := &PeerManager{}

	ret.peers = cmap.New()
	ret.publicToZif = cmap.New()
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

	if err != nil {
		return nil, err
	}

	if pm.socks {
		peer.streams.Socks = true
		peer.streams.SocksPort = pm.socksPort
	}

	err = peer.Connect(addr, pm.localPeer)

	if err != nil {
		return nil, err
	}

	peer.ConnectClient(pm.localPeer)

	pm.SetPeer(peer)

	return peer, nil
}

// Resolved a Zif address into an entry, connects to the peer at the
// PublicAddress in the Entry, then return it. The peer is also stored in a map.
func (lp *LocalPeer) ConnectPeer(addr string) (*Peer, error) {
	var peer *Peer

	if peer = lp.GetPeer(addr); peer != nil {
		return peer, nil
	}

	entry, err := lp.Resolve(addr)

	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	if entry == nil {
		return nil, data.AddressResolutionError{addr}
	}

	// now should have an entry for the peer, connect to it!
	s, _ := entry.Address.String()
	log.Debug("Connecting to ", s)

	peer, err = lp.ConnectPeerDirect(entry.PublicAddress + ":" + strconv.Itoa(entry.Port))

	// Caller can go on to choose a seed to connect to, not quite the end of the
	// world :P
	if err != nil {
		log.WithField("peer", addr).Info("Failed to connect")

		return nil, err
	}

	return peer, nil
}

func (pm *PeerManager) GetPeer(addr string) *Peer {
	peer, has := pm.peers.Get(addr)

	if !has {
		return nil
	}

	return peer.(*Peer)
}

func (pm *PeerManager) SetPeer(p *Peer) {
	s, _ := p.Address().String()
	pm.peers.Set(s, p)

	e, err := p.Entry()

	if err != nil {
		log.Error(err.Error())
		return
	}

	pm.publicToZif.Set(e.PublicAddress, s)
}

func (pm *PeerManager) HandleCloseConnection(addr *dht.Address) {
	s, _ := addr.String()
	pm.peers.Remove(s)
}
