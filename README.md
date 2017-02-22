# Zif

[Whitepaper](https://github.com/zif/whitepaper)

A distributed torrent sharing and indexing network.

**NOTE: I'd prefer if people please refrain from putting this on Hacker News/Reddit/lobste.rs/etc for the moment, it's still buggy and messy and not yet how I want it to be**

Check out the ``develop`` branch for the latest work!

# Why?

I believe in anyone being able to share any information they wish, with anyone - without censorship getting in the way. Bittorrent is an excellent file-transfer protocol, though the torrent/magnet link discovery has been centralised and flawed since its inception.

This is my attempt at a solution.

# What even is this?

- a queryable database of torrents and metadata
- a p2p network for the distribution of torrent info hashes and metadata

Peer discovery is done via an implementation of Kademlia, a DHT similar to the one Bittorrent uses. Peers are assigned addresses similar to Bitcoin addresses, except using Ed25519 and SHA3.

# Sounds cool, when can I use it?

Sometime in the future. As of the time of writing, I have written a DHT for peer resolution, a database system for post storage/indexing, and a protocol for remote searching of posts and mirroring of peer databases. There's also a graphical client in the works, see the ui folder!

It's actually relatively usable at the moment, just needs more testing - the UI also needs work for it to be properly functional.
