package libzif

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/streamrail/concurrent-map"
	data "github.com/zif/zif/libzif/data"
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

	peer, err := cs.LocalPeer.ConnectPeer(p.Address)

	if err != nil {
		return CommandResult{false, nil, err}
	}

	time, err := peer.Ping()

	return CommandResult{err == nil, time.Seconds(), err}
}
func (cs *CommandServer) Announce(a CommandAnnounce) CommandResult {
	var err error

	log.Info("Command: Announce request")

	peer := cs.LocalPeer.GetPeer(a.Address)

	if peer == nil {
		peer, err = cs.LocalPeer.ConnectPeer(a.Address)

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

	peer := cs.LocalPeer.GetPeer(rs.CommandPeer.Address)

	if peer == nil {
		// Remote searching is not allowed to be done on seeds, it has no
		// verification so can be falsified easily. Mirror people, mirror!
		peer, err = cs.LocalPeer.ConnectPeer(rs.CommandPeer.Address)
		if err != nil {
			return CommandResult{false, nil, err}
		}
	}

	posts, stream, err := peer.Search(rs.Query, rs.Page)

	if stream != nil {
		defer stream.Close()
	}

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
func (cs *CommandServer) PeerRecent(pr CommandPeerRecent) CommandResult {
	var err error
	var posts []*data.Post

	log.Info("Command: Peer Recent request")

	if pr.CommandPeer.Address == cs.LocalPeer.Entry.Address.String() {
		posts, err = cs.LocalPeer.Database.QueryRecent(pr.Page)

		return CommandResult{err == nil, posts, err}
	}

	peer := cs.LocalPeer.GetPeer(pr.CommandPeer.Address)
	if peer == nil {
		peer, err = cs.LocalPeer.ConnectPeer(pr.CommandPeer.Address)
		if err != nil {
			return CommandResult{false, nil, err}
		}
	}

	posts, stream, err := peer.Recent(pr.Page)

	if stream != nil {
		defer stream.Close()
	}

	return CommandResult{err == nil, posts, err}
}
func (cs *CommandServer) PeerPopular(pp CommandPeerPopular) CommandResult {
	var err error
	var posts []*data.Post

	log.Info("Command: Peer Popular request")

	if pp.CommandPeer.Address == cs.LocalPeer.Entry.Address.String() {
		posts, err = cs.LocalPeer.Database.QueryPopular(pp.Page)

		return CommandResult{err == nil, posts, err}
	}

	peer := cs.LocalPeer.GetPeer(pp.CommandPeer.Address)
	if peer == nil {
		peer, err = cs.LocalPeer.ConnectPeer(pp.CommandPeer.Address)
		if err != nil {
			return CommandResult{false, nil, err}
		}
	}

	posts, stream, err := peer.Popular(pp.Page)

	if stream != nil {
		defer stream.Close()
	}

	return CommandResult{err == nil, posts, err}
}
func (cs *CommandServer) Mirror(cm CommandMirror) CommandResult {
	var err error

	log.Info("Command: Peer Mirror request")

	peer := cs.LocalPeer.GetPeer(cm.Address)

	if peer == nil {
		peer, err = cs.LocalPeer.ConnectPeer(cm.Address)

		if err != nil {
			return CommandResult{false, nil, err}
		}
	}

	// TODO: make this configurable
	d := fmt.Sprintf("./data/%s", peer.Address().String())
	os.Mkdir(fmt.Sprintf("./data/%s", d), 0777)
	db := data.NewDatabase(d)
	db.Connect()

	cs.LocalPeer.Databases.Set(peer.Address().String(), db)

	progressChan := make(chan int)

	go func() {
		for i := range progressChan {
			log.Debug("Piece, id ", i)
			cs.MirrorProgress.Set(cm.Address, i)
		}
	}()

	_, err = peer.Mirror(db, progressChan)
	if err != nil {
		return CommandResult{false, nil, err}
	}

	// TODO: wjh: is this needed? -poro
	cs.LocalPeer.Databases.Set(peer.Address().String(), db)

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

	post := data.Post{ap.Id, ap.InfoHash, ap.Title, ap.Size, ap.FileCount, ap.Seeders, ap.Leechers, ap.UploadDate, ap.Tags, ap.Meta}

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

	entry, err := cs.LocalPeer.Resolve(cr.Address)

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

	_, err = peer.Bootstrap(cs.LocalPeer.DHT)

	return CommandResult{err == nil, nil, err}
}
func (cs *CommandServer) SelfSuggest(css CommandSuggest) CommandResult {
	completions, err := cs.LocalPeer.SearchProvider.Suggest(cs.LocalPeer.Database, css.Query)

	return CommandResult{err == nil, completions, err}
}
func (cs *CommandServer) SelfSearch(css CommandSelfSearch) CommandResult {
	log.Info("Command: Search request")

	posts, err := cs.LocalPeer.SearchProvider.Search(cs.LocalPeer.Address().String(), cs.LocalPeer.Database, css.Query, css.Page)

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

	ps := make([]*Peer, cs.LocalPeer.Peers.Count()+1)

	ps[0] = &cs.LocalPeer.Peer

	i := 1
	for _, p := range cs.LocalPeer.Peers.Items() {
		ps[i] = p.(*Peer)
		i = i + 1
	}

	return CommandResult{true, ps, nil}
}

func (cs *CommandServer) RequestAddPeer(crap CommandRequestAddPeer) CommandResult {
	log.Info("Command: Request Add Peer request")

	peer, err := cs.LocalPeer.ConnectPeer(crap.Remote)

	if err != nil {
		return CommandResult{true, nil, err}
	}

	_, err = peer.RequestAddPeer(crap.Peer)

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
		value = cs.LocalPeer.Entry.Address.String()
	case "postcount":
		value = strconv.Itoa(cs.LocalPeer.Entry.PostCount)
	case "entry":
		value, _ = cs.LocalPeer.Entry.JsonString()

	default:
		return CommandResult{false, nil, errors.New("Unknown key")}
	}

	return CommandResult{true, value, nil}
}
