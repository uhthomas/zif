package libzif_test

import (
	"fmt"
	"strconv"
	"testing"

	log "github.com/sirupsen/logrus"
)

func CreateLocalPeer(name string, port int) zif.LocalPeer {
	var lp zif.LocalPeer

	lp.GenerateKey()
	lp.Setup()
	lp.RoutingTable.Setup(lp.ZifAddress)

	lp.Entry.Name = name
	lp.Entry.Port = port
	lp.Entry.PublicAddress = "127.0.0.1"
	lp.Entry.Desc = "Decentralize all the things!"
	lp.Entry.PublicKey = lp.PublicKey
	lp.Entry.ZifAddress = lp.ZifAddress

	lp.Database = zif.NewDatabase(fmt.Sprintf("test-%s.db", name))
	lp.Database.Connect()

	lp.SignEntry()

	lp.Listen("0.0.0.0:" + strconv.Itoa(port))

	return lp
}

func BootstrapLocalPeer(lp *zif.LocalPeer, peer *zif.LocalPeer, t *testing.T) {
	p, err := lp.ConnectPeerDirect(peer.Entry.PublicAddress + ":" +
		strconv.Itoa(peer.Entry.Port))

	if err != nil {
		t.Fatal(err.Error())
	}

	p.ConnectClient(lp)

	stream, err := p.Bootstrap(&lp.RoutingTable)

	if err != nil {
		t.Fatal(err.Error())
	}

	stream.Close()

}

// Does a *simple* test of announcing.
// Further testig with a much larger number of networked peers (over the internet)
// is definitely needed.
func TestLocalPeerAnnounce(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	// both peers know this one
	lp_initial := CreateLocalPeer("initial", 5050)

	// this peers announces itself
	lp_announcer := CreateLocalPeer("announcer", 5051)

	// This peer should be able to resolve the announcers address after it announces
	lp_test := CreateLocalPeer("test", 5052)

	defer lp_initial.Close()
	defer lp_announcer.Close()
	defer lp_test.Close()

	BootstrapLocalPeer(&lp_announcer, &lp_initial, t)
	BootstrapLocalPeer(&lp_test, &lp_initial, t)

	// announce the test to the initial peer
	peer, err := lp_test.ConnectPeer(lp_initial.ZifAddress.Encode())

	if err != nil {
		t.Fatal(err.Error())
	}

	peer.Announce(&lp_test)
	////////////////////////////////////////////

	if lp_initial.RoutingTable.NumPeers() != 2 {
		t.Fatal("Failed to store announcements properly")
	}

	// announce the announcer to the initial peer
	peer, err = lp_announcer.ConnectPeer(lp_initial.ZifAddress.Encode())

	if err != nil {
		t.Fatal(err.Error())
	}

	peer.Announce(&lp_announcer)
	////////////////////////////////////////////

	if lp_initial.RoutingTable.NumPeers() <= 1 {
		t.Fatal("Failed to store announcements properly")
	}

	// block until the test peer recieved a msg (should be an announce forward)
	<-lp_test.MsgChan

	if !lp_test.RoutingTable.FindClosest(lp_initial.ZifAddress, 1)[0].
		ZifAddress.Equals(&lp_initial.ZifAddress) {
		t.Fatal("Announce forwarding failed")
	}
}

func TestLocalPeerPosts(t *testing.T) {
	source, _ := zif.CryptoRandBytes(20)
	// the remote we are requesting posts from
	lp_remote := CreateLocalPeer("remote", 5053)
	lp_requester := CreateLocalPeer("requester", 5054)

	defer lp_remote.Close()
	defer lp_requester.Close()

	BootstrapLocalPeer(&lp_requester, &lp_remote, t)

	arch := zif.NewPost(ArchInfoHash, "Arch Linux 2015-09-03", 100, 10, 1472860800, source)
	ubuntu := zif.NewPost(UbuntuInfoHash, "Ubuntu Linux 16.04.1", 101, 9, 1472860800, source)

	lp_remote.AddPost(arch, false)
	lp_remote.AddPost(ubuntu, false)
	lp_remote.Database.GenerateFts(0)

	peer, err := lp_requester.ConnectPeer(lp_remote.ZifAddress.Encode())

	if err != nil {
		t.Fatal(err.Error())
	}

	posts, stream, err := peer.Search("linux", 0)
	defer stream.Close()

	if err != nil {
		t.Fatal(err.Error())
	}

	if len(posts) != 2 {
		t.Fatal("Incorrect post count returned")
	}

	if posts[0].InfoHash != UbuntuInfoHash || posts[1].InfoHash != ArchInfoHash {
		t.Error("Remote post search failed")
	}
}

func TestLocalPeerRecent(t *testing.T) {
	source, _ := zif.CryptoRandBytes(20)
	lp_remote := CreateLocalPeer("remote", 5055)
	lp_requester := CreateLocalPeer("requester", 5056)

	defer lp_remote.Close()
	defer lp_requester.Close()

	BootstrapLocalPeer(&lp_requester, &lp_remote, t)

	arch := zif.NewPost(ArchInfoHash, "Arch Linux 2015-09-03", 100, 10, 1472860800, source)
	ubuntu := zif.NewPost(UbuntuInfoHash, "Ubuntu Linux 16.04.1", 101, 9, 1472860800, source)

	lp_remote.AddPost(arch, false)
	lp_remote.AddPost(ubuntu, false)
	lp_remote.Database.GenerateFts(0)

	peer, err := lp_requester.ConnectPeer(lp_remote.ZifAddress.Encode())

	if err != nil {
		t.Fatal(err.Error())
	}

	posts, stream, err := peer.Recent(0)
	defer stream.Close()

	if err != nil {
		t.Fatal(err.Error())
	}

	if len(posts) != 2 {
		t.Fatal("Incorrect post count returned")
	}

	if posts[0].InfoHash != UbuntuInfoHash || posts[1].InfoHash != ArchInfoHash {
		t.Error("Remote post search failed")
	}
}
