package jobs

import (
	"time"

	"github.com/zif/zif/libzif/common"
	"github.com/zif/zif/libzif/dht"
)

const ExploreSleepTime = time.Minute * 2
const ExploreBufferSize = 100

// This job runs every two minutes, and tries to build the netdb with as many
// entries as it possibly can
func ExploreJob(in <-chan interface{}, data ...interface{}) chan<- interface{} {
	ret := make(chan<- interface{}, ExploreBufferSize)
	connector := data[0].(common.ConnectPeer)
	me := data[1].(dht.Address)

	go func() {
		for i := range in {
			explorePeer(i.(dht.Address), me, ret, connector)

			time.Sleep(ExploreSleepTime)
		}

	}()

	return ret
}

func explorePeer(addr dht.Address, me dht.Address, ret chan<- interface{}, connectPeer common.ConnectPeer) error {
	peer, err := connectPeer(addr.String())

	if err != nil {
		return err
	}

	client, closestToMe, err := peer.FindClosest(addr.String())

	if err != nil {
		return err
	}

	client.Close()

	for _, i := range closestToMe {
		ret <- i
	}

	randAddr, err := dht.RandomAddress()

	if err != nil {
		return err
	}

	client, closest, err := peer.FindClosest(randAddr.String())

	if err != nil {
		return err
	}
	client.Close()

	for _, i := range closest {
		ret <- i
	}

	return nil
}
