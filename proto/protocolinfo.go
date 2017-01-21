// Stores things like message codes, etc.

package proto

var (
	// Protocol header, so we know this is a zif client.
	// Version should follow.
	ProtoZif     int16 = 0x7a66
	ProtoVersion int16 = 0x0000

	ProtoHeader = 0x0000

	// inform a peer on the status of the latest request
	ProtoOk        = 0x0001
	ProtoNo        = 0x0002
	ProtoTerminate = 0x0003
	ProtoCookie    = 0x0004
	ProtoSig       = 0x0005
	ProtoPing      = 0x0006
	ProtoPong      = 0x0007
	ProtoDone      = 0x0008

	ProtoSearch  = 0x0101 // Request a search
	ProtoRecent  = 0x0102 // Request recent posts
	ProtoPopular = 0x0103 // Request popular posts

	// Request a signed hash list
	// The content field should contain the bytes for a Zif address.
	// This is the peer we are requesting a hash list for.
	ProtoRequestHashList = 0x0104
	ProtoRequestPiece    = 0x0105
	// Requests that this peer be added to the remotes Peers slice for a given
	// entry. This must be called at least once every hour to ensure that the peer
	// stays registered as a seed, otherwise it is culled.
	// TODO: Look into how Bittorrent trackers keep peer lists up to date properly.
	ProtoRequestAddPeer = 0x0106

	ProtoEntry    = 0x0200 // An individual DHT entry in Content
	ProtoPosts    = 0x0201 // A list of posts in Content
	ProtoHashList = 0x0202
	ProtoPiece    = 0x0203
	ProtoPost     = 0x0204

	ProtoDhtQuery       = 0x0300
	ProtoDhtAnnounce    = 0x0301
	ProtoDhtFindClosest = 0x0302
)
