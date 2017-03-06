package zif

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"runtime/pprof"

	"github.com/zif/zif/data"
	"github.com/zif/zif/dht"
	"github.com/zif/zif/util"

	log "github.com/sirupsen/logrus"
	"github.com/streamrail/concurrent-map"
)

// Command server type

type CommandServer struct {
	LocalPeer *LocalPeer

	// Piece count for ongoing mirrors
	MirrorProgress cmap.ConcurrentMap
}

func NewCommandServer(lp *LocalPeer) *CommandServer {
	ret := &CommandServer{
		LocalPeer:      lp,
		MirrorProgress: cmap.New(),
	}

	return ret
}

// Command functions

func (cs *CommandServer) Ping(p CommandPing) CommandResult {
	log.Info("Command: Ping request")

	address, err := dht.DecodeAddress(p.Address)

	if err != nil {
		return CommandResult{false, nil, err}
	}

	peer, _, err := cs.LocalPeer.ConnectPeer(address)

	if err != nil {
		return CommandResult{false, nil, err}
	}

	time, err := peer.Ping(time.Second * 10)

	return CommandResult{err == nil, time.Seconds(), err}
}
func (cs *CommandServer) Announce(a CommandAnnounce) CommandResult {
	var err error

	log.Info("Command: Announce request")

	address, err := dht.DecodeAddress(a.Address)

	if err != nil {
		return CommandResult{false, nil, err}
	}

	peer := cs.LocalPeer.GetPeer(address)

	if peer == nil {
		peer, _, err = cs.LocalPeer.ConnectPeer(address)

		if err != nil {
			return CommandResult{false, nil, err}
		}
	}

	if err != nil {
		return CommandResult{false, nil, err}
	}

	err = peer.Announce(cs.LocalPeer)

	return CommandResult{err == nil, nil, err}
}
func (cs *CommandServer) RSearch(rs CommandRSearch) CommandResult {
	var err error

	log.Info("Command: Peer Remote Search request")

	address, err := dht.DecodeAddress(rs.Address)

	if err != nil {
		return CommandResult{false, nil, err}
	}

	peer := cs.LocalPeer.GetPeer(address)

	if peer == nil {
		// Remote searching is not allowed to be done on seeds, it has no
		// verification so can be falsified easily. Mirror people, mirror!
		peer, _, err = cs.LocalPeer.ConnectPeer(address)
		if err != nil {
			return CommandResult{false, nil, err}
		}
	}

	posts, err := peer.Search(rs.Query, rs.Page)

	return CommandResult{err == nil, posts, err}
}
func (cs *CommandServer) PeerSearch(ps CommandPeerSearch) CommandResult {
	var err error

	log.Info("Command: Peer Search request")

	if !cs.LocalPeer.Databases.Has(ps.CommandPeer.Address) {
		return cs.RSearch(CommandRSearch{ps.CommandPeer, ps.Query, ps.Page})
	}

	db, _ := cs.LocalPeer.Databases.Get(ps.CommandPeer.Address)

	posts, err := cs.LocalPeer.SearchProvider.Search(ps.CommandPeer.Address, db.(*data.Database), ps.Query, ps.Page)

	return CommandResult{err == nil, posts, err}
}

func (cs *CommandServer) EntrySearch(ps CommandSearchEntry) CommandResult {
	var err error

	log.Info("Command: Entry Search request")

	addresses, err := cs.LocalPeer.DHT.SearchEntries(ps.Name, ps.Desc, ps.Page)

	return CommandResult{err == nil, addresses, err}
}

func (cs *CommandServer) PeerRecent(pr CommandPeerRecent) CommandResult {
	var err error
	var posts []*data.Post

	log.Info("Command: Peer Recent request")

	if pr.CommandPeer.Address == cs.LocalPeer.Address().StringOr("") {
		posts, err = cs.LocalPeer.Database.QueryRecent(pr.Page)

		return CommandResult{err == nil, posts, err}
	}

	address, err := dht.DecodeAddress(pr.Address)

	if err != nil {
		return CommandResult{false, nil, err}
	}

	peer := cs.LocalPeer.GetPeer(address)

	if peer == nil {
		peer, _, err = cs.LocalPeer.ConnectPeer(address)
		if err != nil {
			return CommandResult{false, nil, err}
		}
	}

	posts, err = peer.Recent(pr.Page)

	return CommandResult{err == nil, posts, err}
}
func (cs *CommandServer) PeerPopular(pp CommandPeerPopular) CommandResult {
	var err error
	var posts []*data.Post

	log.Info("Command: Peer Popular request")

	if pp.CommandPeer.Address == cs.LocalPeer.Address().StringOr("") {
		posts, err = cs.LocalPeer.Database.QueryPopular(pp.Page)

		return CommandResult{err == nil, posts, err}
	}

	address, err := dht.DecodeAddress(pp.Address)

	if err != nil {
		return CommandResult{false, nil, err}
	}

	peer := cs.LocalPeer.GetPeer(address)

	if peer == nil {
		peer, _, err = cs.LocalPeer.ConnectPeer(address)
		if err != nil {
			return CommandResult{false, nil, err}
		}
	}

	posts, err = peer.Popular(pp.Page)

	return CommandResult{err == nil, posts, err}
}
func (cs *CommandServer) Mirror(cm CommandMirror) CommandResult {
	var err error

	log.Info("Command: Peer Mirror request")

	address, err := dht.DecodeAddress(cm.Address)

	if err != nil {
		return CommandResult{false, nil, err}
	}

	mirroring, err := cs.LocalPeer.Resolve(address)

	if err != nil {
		return CommandResult{false, nil, err}
	}

	peer := cs.LocalPeer.GetPeer(address)

	if peer == nil {
		var entry *dht.Entry
		peer, entry, err = cs.LocalPeer.ConnectPeer(address)

		if err != nil {
			if err == PeerUnreachable {

				// first up, check if we have an entry for the peer
				if len(entry.Seeds) == 0 {
					// if not, not much we can do :(
					return CommandResult{false, nil, err}
				}

				// balances load amongst all seeds
				util.ShuffleBytes(entry.Seeds)

				// Keep picking seeds until one connects
				for _, i := range entry.Seeds {
					addr := &dht.Address{Raw: i}

					if addr.Equals(cs.LocalPeer.Address()) {
						continue
					}

					peer, _, err = cs.LocalPeer.ConnectPeer(*addr)

					if err != nil || peer == nil {
						continue
					}

					// make sure the correct values are chosen when mirroring
					// peers act a little differently when seeding for another
					peer.seed = true
					peer.seedFor = entry
				}

			} else {
				return CommandResult{false, nil, err}
			}
		}
	}

	if peer == nil {
		return CommandResult{false, nil, PeerUnreachable}
	}

	// TODO: make this configurable
	d := fmt.Sprintf("./data/%s", mirroring.Address.StringOr(""))

	os.Mkdir(d, 0777)

	db := data.NewDatabase(fmt.Sprintf("%s/posts.db", d))
	db.Connect()

	cs.LocalPeer.Databases.Set(peer.Address().StringOr(""), db)

	progressChan := make(chan int)

	go func() {
		for i := range progressChan {
			cs.MirrorProgress.Set(cm.Address, i)
		}
	}()

	err = peer.Mirror(db, *cs.LocalPeer.Address(), progressChan)
	if err != nil {
		return CommandResult{false, nil, err}
	}

	return CommandResult{true, nil, nil}
}

func (cs *CommandServer) GetMirrorProgress(cmp CommandMirrorProgress) CommandResult {
	if !cs.MirrorProgress.Has(cmp.Address) {
		return CommandResult{false, nil, errors.New("Mirror not in progress")}
	}

	progress, _ := cs.MirrorProgress.Get(cmp.Address)

	return CommandResult{true, progress.(int), nil}
}

func (cs *CommandServer) PeerIndex(ci CommandPeerIndex) CommandResult {
	var err error

	log.Info("Command: Peer Index request")

	if !cs.LocalPeer.Databases.Has(ci.CommandPeer.Address) {
		return CommandResult{false, nil, errors.New("Peer database not loaded.")}
	}

	db, _ := cs.LocalPeer.Databases.Get(ci.CommandPeer.Address)
	err = db.(*data.Database).GenerateFts(int64(ci.Since))

	return CommandResult{err == nil, nil, err}
}

func (cs *CommandServer) AddPost(ap CommandAddPost) CommandResult {
	log.Info("Command: Add Post request")

	post := data.Post{
		Id:         ap.Id,
		InfoHash:   ap.InfoHash,
		Title:      ap.Title,
		Size:       ap.Size,
		FileCount:  ap.FileCount,
		Seeders:    ap.Seeders,
		Leechers:   ap.Leechers,
		UploadDate: ap.UploadDate,
		Tags:       ap.Tags,
		Meta:       ap.Meta,
	}

	id, err := cs.LocalPeer.AddPost(post, false)

	if err != nil {
		return CommandResult{false, nil, err}
	}

	if ap.Index {
		cs.LocalPeer.Database.GenerateFts(id - 1)
	}

	return CommandResult{true, id, nil}
}
func (cs *CommandServer) SelfIndex(ci CommandSelfIndex) CommandResult {
	log.Info("Command: FTS Index request")

	err := cs.LocalPeer.Database.GenerateFts(int64(ci.Since))

	return CommandResult{err == nil, nil, err}
}
func (cs *CommandServer) Resolve(cr CommandResolve) CommandResult {
	log.Info("Command: Resolve request")

	address, err := dht.DecodeAddress(cr.Address)

	if err != nil {
		return CommandResult{false, nil, err}
	}

	entry, err := cs.LocalPeer.Resolve(address)

	if err != nil {
		return CommandResult{false, nil, err}
	}

	// forces the address to generate its encoded value, so this is then
	// available in JSON.
	entry.Address.String()

	return CommandResult{err == nil, entry, err}
}
func (cs *CommandServer) Bootstrap(cb CommandBootstrap) CommandResult {
	log.Info("Command: Bootstrap request")

	addrnport := strings.Split(cb.Address, ":")

	host := addrnport[0]
	var port string
	if len(addrnport) == 1 {
		port = "5050" // TODO: make this configurable
	} else {
		port = addrnport[1]
	}

	peer, err := cs.LocalPeer.ConnectPeerDirect(host + ":" + port)
	if err != nil {
		return CommandResult{false, nil, err}
	}

	err = peer.Bootstrap(cs.LocalPeer.DHT)

	return CommandResult{err == nil, nil, err}
}
func (cs *CommandServer) SelfSuggest(css CommandSuggest) CommandResult {
	completions, err := cs.LocalPeer.SearchProvider.Suggest(cs.LocalPeer.Database, css.Query)

	return CommandResult{err == nil, completions, err}
}
func (cs *CommandServer) SelfSearch(css CommandSelfSearch) CommandResult {
	log.Info("Command: Search request")

	posts, err := cs.LocalPeer.SearchProvider.Search(cs.LocalPeer.Address().StringOr(""),
		cs.LocalPeer.Database, css.Query, css.Page)

	return CommandResult{err == nil, posts, err}
}
func (cs *CommandServer) SelfRecent(cr CommandSelfRecent) CommandResult {
	log.Info("Command: Recent request")

	posts, err := cs.LocalPeer.Database.QueryRecent(cr.Page)

	return CommandResult{err == nil, posts, err}
}
func (cs *CommandServer) SelfPopular(cp CommandSelfPopular) CommandResult {
	log.Info("Command: Popular request")

	posts, err := cs.LocalPeer.Database.QueryPopular(cp.Page)

	return CommandResult{err == nil, posts, err}
}
func (cs *CommandServer) AddMeta(cam CommandAddMeta) CommandResult {
	log.Info("Command: Add Meta request")

	err := cs.LocalPeer.Database.AddMeta(cam.CommandMeta.PId, cam.Value)

	return CommandResult{err == nil, nil, err}
}
func (cs *CommandServer) SaveCollection(csc CommandSaveCollection) CommandResult {
	log.Info("Command: Save Collection request")

	// TODO: make this configurable
	cs.LocalPeer.Collection.Save("./data/collection.dat")

	return CommandResult{true, nil, nil}
}
func (cs *CommandServer) RebuildCollection(crc CommandRebuildCollection) CommandResult {
	var err error

	log.Info("Command: Rebuild Collection request")

	cs.LocalPeer.Collection, err = data.CreateCollection(cs.LocalPeer.Database, 0, data.PieceSize)
	return CommandResult{err == nil, nil, err}
}
func (cs *CommandServer) Peers(cp CommandPeers) CommandResult {
	log.Info("Command: Peers request")

	ps := make([]*dht.Entry, cs.LocalPeer.PeerCount()+1)
	var err error

	ps[0], err = cs.LocalPeer.Peer.Entry()

	if err != nil {
		return CommandResult{false, nil, err}
	}

	i := 1
	for _, p := range cs.LocalPeer.Peers() {
		ps[i], err = p.Entry()

		if err != nil {
			return CommandResult{false, nil, err}
		}

		i = i + 1
	}

	return CommandResult{true, ps, nil}
}

func (cs *CommandServer) RequestAddPeer(crap CommandRequestAddPeer) CommandResult {
	log.Info("Command: Request Add Peer request")

	address, err := dht.DecodeAddress(crap.Peer)

	if err != nil {
		return CommandResult{true, nil, err}
	}

	peer, _, err := cs.LocalPeer.ConnectPeer(address)

	if err != nil {
		return CommandResult{true, nil, err}
	}

	entry, err := cs.LocalPeer.QueryEntry(address)

	if err != nil {
		return CommandResult{true, nil, err}
	}

	err = peer.RequestAddPeer(*entry)

	return CommandResult{err == nil, nil, err}
}

// Set a value in the localpeer entry
func (cs *CommandServer) LocalSet(cls CommandLocalSet) CommandResult {

	switch strings.ToLower(cls.Key) {
	case "name":
		cs.LocalPeer.Entry.Name = cls.Value
	case "desc":
		cs.LocalPeer.Entry.Desc = cls.Value
	case "public":
		cs.LocalPeer.Entry.PublicAddress = cls.Value

	default:
		return CommandResult{false, nil, errors.New("Unknown key")}
	}

	cs.LocalPeer.SignEntry()
	err := cs.LocalPeer.SaveEntry()

	return CommandResult{err == nil, nil, err}
}

func (cs *CommandServer) LocalGet(clg CommandLocalGet) CommandResult {
	log.Info("Command: LocalGet")
	value := ""

	switch strings.ToLower(clg.Key) {
	case "name":
		value = cs.LocalPeer.Entry.Name
	case "desc":
		value = cs.LocalPeer.Entry.Desc
	case "public":
		value = cs.LocalPeer.Entry.PublicAddress
	case "zif":
		value, _ = cs.LocalPeer.Entry.Address.String()
	case "postcount":
		value = strconv.Itoa(cs.LocalPeer.Entry.PostCount)
	case "entry":
		value, _ = cs.LocalPeer.Entry.EncodeString()

	default:
		return CommandResult{false, nil, errors.New("Unknown key")}
	}

	return CommandResult{true, value, nil}
}

func (cs *CommandServer) Explore() CommandResult {
	err := cs.LocalPeer.StartExploring()

	return CommandResult{err == nil, nil, err}
}

func (cs *CommandServer) AddressEncode(ce CommandAddressEncode) CommandResult {
	log.Info("Encode request")
	address := &dht.Address{Raw: ce.Raw}

	decoded, err := address.String()

	return CommandResult{err == nil, decoded, err}
}

func (cs *CommandServer) StartCpuProfile(cf CommandFile) CommandResult {
	f, err := os.Create(cf.File)

	if err != nil {
		return CommandResult{false, nil, err}
	}

	err = pprof.StartCPUProfile(f)

	return CommandResult{err == nil, nil, err}
}

func (cs *CommandServer) StopCpuProfile() {
	pprof.StopCPUProfile()
}

func (cs *CommandServer) MemProfile(cf CommandFile) CommandResult {
	f, err := os.Create(cf.File)

	if err != nil {
		return CommandResult{false, nil, err}
	}

	err = pprof.WriteHeapProfile(f)

	return CommandResult{err == nil, nil, err}
}

func (cs *CommandServer) SetSeedLeech(csl CommandSetSeedLeech) CommandResult {
	err := cs.LocalPeer.Database.SetLeechers(csl.Id, csl.Leechers)

	if err != nil {
		return CommandResult{false, nil, err}
	}

	err = cs.LocalPeer.Database.SetSeeders(csl.Id, csl.Seeders)

	return CommandResult{err == nil, nil, err}
}
