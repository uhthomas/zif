package jobs

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zif/zif/common"
	"github.com/zif/zif/dht"
)

const ExploreSleepTime = time.Minute * 2
const ExploreBufferSize = 100

// This job runs every two minutes, and tries to build the netdb with as many
// entries as it possibly can
func ExploreJob(in chan dht.KeyValue, data ...interface{}) <-chan dht.KeyValue {
	ret := make(chan dht.KeyValue, ExploreBufferSize)

	connector := data[0].(func(string) (interface{}, error))
	me := data[1].(dht.Address)
	seed := data[2].(func(ret chan dht.KeyValue))

	ticker := time.NewTicker(ExploreSleepTime)

	go exploreTick(in, ret, me, connector, seed)

	go func() {
		for _ = range ticker.C {
			go exploreTick(in, ret, me, connector, seed)
		}

	}()

	return ret
}

func exploreTick(in chan dht.KeyValue, ret chan dht.KeyValue, me dht.Address, connector common.ConnectPeer, seed func(chan dht.KeyValue)) {
	i := <-in
	s, _ := i.Key().String()

	if i.Key().Equals(&me) {
		return
	}

	log.WithField("peer", s).Info("Exploring")

	if err := explorePeer(*i.Key(), me, ret, connector); err != nil {
		log.Info(err.Error())
	}

	if len(in) == 0 {
		seed(in)
		log.Info("Seeding peer explore")
	}
}

func explorePeer(addr dht.Address, me dht.Address, ret chan<- dht.KeyValue, connectPeer common.ConnectPeer) error {
	s, _ := addr.String()
	me_s, _ := me.String()
	peer, err := connectPeer(s)
	p := peer.(common.Peer)

	if err != nil {
		return err
	}

	closestToMe, err := p.FindClosest(me_s)

	if err != nil {
		return err
	}

	for _, i := range closestToMe {
		ret <- *i
	}

	randAddr, err := dht.RandomAddress()

	if err != nil {
		return err
	}

	rs, _ := randAddr.String()
	closest, err := p.FindClosest(rs)

	if err != nil {
		return err
	}

	for _, i := range closest {
		if !i.Key().Equals(&me) {
			ret <- *i
		}
	}

	return nil
}
