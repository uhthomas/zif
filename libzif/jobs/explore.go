package jobs

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zif/zif/libzif/common"
	"github.com/zif/zif/libzif/dht"
)

const ExploreSleepTime = time.Minute * 2
const ExploreBufferSize = 100

// This job runs every two minutes, and tries to build the netdb with as many
// entries as it possibly can
func ExploreJob(in <-chan dht.KeyValue, data ...interface{}) <-chan dht.KeyValue {
	ret := make(chan dht.KeyValue, ExploreBufferSize)
	connector := data[0].(func(string) (interface{}, error))
	me := data[1].(dht.Address)

	go func() {
		for i := range in {
			s, _ := i.Key().String()
			log.WithField("peer", s).Info("Exploring")

			if err := explorePeer(*i.Key(), me, ret, connector); err != nil {
				log.Info(err.Error())
				continue
			}

			time.Sleep(ExploreSleepTime)
		}

	}()

	return ret
}

func explorePeer(addr dht.Address, me dht.Address, ret chan<- dht.KeyValue, connectPeer common.ConnectPeer) error {
	s, _ := addr.String()
	peer, err := connectPeer(s)
	p := peer.(common.Peer)

	if err != nil {
		return err
	}

	client, closestToMe, err := p.FindClosest(s)

	if err != nil {
		return err
	}

	client.Close()

	for _, i := range closestToMe {
		ret <- *i
	}

	randAddr, err := dht.RandomAddress()

	if err != nil {
		return err
	}

	rs, _ := randAddr.String()
	client, closest, err := p.FindClosest(rs)

	if err != nil {
		return err
	}
	client.Close()

	for _, i := range closest {
		if !i.Key().Equals(&me) {
			ret <- *i
		}
	}

	return nil
}
