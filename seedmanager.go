package zif

import (
	"bytes"
	"time"

	"github.com/zif/zif/dht"
	"github.com/zif/zif/proto"
	"github.com/zif/zif/util"

	log "github.com/sirupsen/logrus"
)

const SeedSearchFrequency = time.Minute * 5

// brought into it's own type to track seed data, and manage it.
// works for all peers that we are seeding, including the localpeer.
// could one day be extended as a "gossip" protocol for stuff like comments,
// methinks.

type SeedManager struct {
	// the localpeer, allows the struct to make connections, etc
	lp *LocalPeer

	// the address we are tracking seeds for
	track dht.Address
	entry *proto.Entry
	Close chan bool
}

func NewSeedManager(track dht.Address, lp *LocalPeer) (*SeedManager, error) {
	ret := SeedManager{
		lp:    lp,
		Close: make(chan bool),
	}

	entry, err := lp.QueryEntry(track)

	if err != nil {
		return nil, err
	}

	ret.entry = entry
	ret.track = track

	return &ret, nil
}

func (sm *SeedManager) Start() {
	log.WithField("peer", sm.track.StringOr("")).Info("Starting seed manager")
	go sm.findSeeds()
}

// queries all seeds to see if we can find new seeds
func (sm *SeedManager) findSeeds() {
	ticker := time.NewTicker(SeedSearchFrequency)

	find := func() {
		entry, err := sm.lp.QueryEntry(sm.track)

		if err != nil {
			log.Error(err.Error())
			return
		}

		sm.entry = entry

		log.Info("Searching for new seeds")
		for _, i := range sm.entry.Seeds {
			addr := dht.Address{Raw: i}

			if addr.Equals(sm.lp.Address()) {
				continue
			}

			s, err := addr.String()
			if err != nil {
				continue
			}

			peer, entry, err := sm.lp.ConnectPeer(addr)

			if err != nil {
				log.Error(err.Error())
				continue
			}

			es, err := entry.Address.String()

			if err != nil {
				log.Error(err.Error())
				continue
			}

			log.WithField("peer", es).Info("Querying for seeds")

			qResultVerifiable, err := peer.Query(sm.entry.Address)
			if err != nil {
				continue
			}

			qResult := qResultVerifiable.(*proto.Entry)

			result := util.SliceDiff(sm.entry.Seeds, qResult.Seeds)

			// make sure all these seeds actually link back! Otherwise they could
			// be fakes
			for n, i := range result {
				seedAddress := dht.Address{Raw: i}

				entry, err := sm.lp.Resolve(seedAddress)

				// nope, we won't be adding this one
				if err != nil {
					if n >= len(result)-1 {
						result = result[:n]
					} else {
						result = append(result[:n], result[n+1:]...)
					}
					result = append(result[:n], result[n+1:]...)
					continue
				}

				// check if the entry has registered itself as a seeder

				found := false
				for _, j := range entry.Seeding {
					if bytes.Equal(sm.track.Raw, j) {
						found = true
						break
					}
				}

				if !found {
					if n >= len(result)-1 {
						result = result[:n]
					} else {
						result = append(result[:n], result[n+1:]...)
					}
					continue
				}
			}

			if len(result) > 0 {
				sm.entry.Seeds = append(sm.entry.Seeds, result...)
				encoded, err := sm.entry.Encode()

				if err != nil {
					continue
				}

				log.WithField("peer", s).Info("Found new seeds")
				sm.lp.DHT.Insert(dht.NewKeyValue(sm.entry.Address, encoded))

			}
		}
	}

	find()

	for {
		select {
		case _ = <-ticker.C:
			find()
		case _ = <-sm.Close:
			return
		}
	}
}
