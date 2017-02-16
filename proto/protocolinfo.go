// Stores things like message codes, etc.

package proto

var (
	// Protocol header, so we know this is a zif client.
	// Version should follow.
	ProtoZif     int16 = 0x7a66
	ProtoVersion int16 = 0x0000

	ProtoHeader = "header"
	ProtoCap    = ":ap"

	// inform a peer on the status of the latest request
	ProtoOk        = "ok"
	ProtoNo        = "no"
	ProtoTerminate = "term"
	ProtoCookie    = "cookie"
	ProtoSig       = "sig"
	ProtoDone      = "done"

	ProtoSearch  = "search"  // Request a search
	ProtoRecent  = "recent"  // Request recent posts
	ProtoPopular = "popular" // Request popular posts

	// Request a signed hash list
	// The content field should contain the bytes for a Zif address.
	// This is the peer we are requesting a hash list for.
	ProtoRequestHashList = "req.hashlist"
	ProtoRequestPiece    = "req.piece"
	// Requests that this peer be added to the remotes Peers slice for a given
	// entry. This must be called at least once every hour to ensure that the peer
	// stays registered as a seed, otherwise it is culled.
	// TODO: Look into how Bittorrent trackers keep peer lists up to date properly.
	ProtoRequestAddPeer = "req.addpeer"

	ProtoPosts    = "posts" // A list of posts in Content
	ProtoHashList = "hashlist"

	ProtoDhtEntry       = "dht.entry" // An individual DHT entry in Content
	ProtoDhtQuery       = "dht.query"
	ProtoDhtAnnounce    = "dht.announce"
	ProtoDhtFindClosest = "dht.findclosest"
)
