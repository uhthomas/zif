package jobs

import (
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zif/zif/common"
	"github.com/zif/zif/dht"
	"github.com/zif/zif/proto"
)

const ExploreFrequency = time.Minute * 2
const ExploreBufferSize = 100

// This job runs every two minutes, and tries to build the netdb with as many
// entries as it possibly can
func ExploreJob(in chan proto.Entry, data ...interface{}) <-chan proto.Entry {
	ret := make(chan proto.Entry, ExploreBufferSize)

	connector := data[0].(func(dht.Address) (interface{}, error))
	me := data[1].(dht.Address)
	seed := data[2].(func(ret chan proto.Entry))

	ticker := time.NewTicker(ExploreFrequency)

	go exploreTick(in, ret, me, connector, seed)

	go func() {
		for _ = range ticker.C {
			go exploreTick(in, ret, me, connector, seed)
		}

	}()

	return ret
}

func exploreTick(in chan proto.Entry, ret chan proto.Entry, me dht.Address, connector common.ConnectPeer, seed func(chan proto.Entry)) {
	i := <-in
	s, _ := i.Address.String()

	if i.Address.Equals(&me) {
		return
	}

	log.WithField("peer", s).Info("Exploring")

	if err := explorePeer(i.Address, me, ret, connector); err != nil {
		log.Error(err.Error())
	}

	if len(in) == 0 {
		seed(in)
		log.Info("Seeding peer explore")
	}
}

func explorePeer(addr dht.Address, me dht.Address, ret chan<- proto.Entry, connectPeer common.ConnectPeer) error {
	peer, err := connectPeer(addr)
	p := peer.(common.Peer)

	if err != nil {
		return err
	}

	self, err := p.Query(addr)

	if err != nil {
		return err
	}

	if self == nil {
		return errors.New("Failure to connect")
	}

	ret <- *self.(*proto.Entry)

	closestToMe, err := p.FindClosest(me)

	if err != nil {
		return err
	}

	for _, i := range closestToMe {
		ret <- *(i.(*proto.Entry))
	}

	randAddr, err := dht.RandomAddress()

	if err != nil {
		return err
	}

	closest, err := p.FindClosest(*randAddr)

	if err != nil {
		return err
	}

	for _, i := range closest {
		if !i.(*proto.Entry).Address.Equals(&me) {
			ret <- *(i.(*proto.Entry))
		}
	}

	return nil
}
