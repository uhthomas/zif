package zif

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/zif/zif/data"
)

// Command input types

type CommandPeer struct {
	Address string `json:"address"`
}

type CommandPing CommandPeer
type CommandAnnounce CommandPeer

type CommandRequestAddPeer struct {
	// The peer to send the request to
	Remote string `json:"remote"`
	// The peer we wish to be registered as a peer for
	Peer string `json:"peer"`
}

type CommandRSearch struct {
	CommandPeer
	Query string `json:"query"`
	Page  int    `json:"page"`
}
type CommandPeerSearch CommandRSearch
type CommandPeerRecent struct {
	CommandPeer
	Page int `json:"page"`
}
type CommandPeerPopular CommandPeerRecent
type CommandMirror CommandPeer
type CommandMirrorProgress CommandPeer
type CommandPeerIndex struct {
	CommandPeer
	Since int `json:"since"`
}

type CommandMeta struct {
	PId int `json:"pid"`
}

type CommandAddPost struct {
	data.Post
	Index bool
}
type CommandSelfIndex struct {
	Since int `json:"since"`
}
type CommandResolve CommandPeer
type CommandBootstrap CommandPeer

type CommandSuggest struct {
	Query string `json:"query"`
}

type CommandSelfSearch struct {
	CommandSuggest
	Page int `json:"page"`
}
type CommandSelfRecent struct {
	Page int `json:"page"`
}
type CommandSelfPopular CommandSelfRecent
type CommandAddMeta struct {
	CommandMeta
	Value string `json:"value"`
}
type CommandGetMeta CommandMeta
type CommandSaveCollection interface{}
type CommandRebuildCollection interface{}
type CommandPeers interface{}
type CommandSaveRoutingTable interface{}

// Used for setting values in the localpeer entry
type CommandLocalSet struct {
	Key   string `json:"key"`
	Value string `json:"key"`
}

type CommandLocalGet struct {
	Key string `json:"key"`
}

type CommandAddressEncode struct {
	Raw []byte `json:"raw"`
}

// Command output types

type CommandResult struct {
	IsOK   bool        `json:"status"`
	Result interface{} `json:"value"`
	Error  error       `json:"err"`
}

func (cr *CommandResult) WriteJSON(w io.Writer) {
	e := json.NewEncoder(w)

	if cr.IsOK {
		if cr.Result == nil {
			e.Encode(struct {
				Status string `json:"status"`
			}{"ok"})
		} else {
			e.Encode(struct {
				Status string      `json:"status"`
				Value  interface{} `json:"value"`
			}{"ok", cr.Result})
		}
	} else {
		if cr.Error == nil {
			cr.Error = errors.New("Something bad happened, but we don't know what, which makes the fact much worse.")
		}

		e.Encode(struct {
			Status string `json:"status"`
			Error  string `json:"err"`
		}{"err", cr.Error.Error()})
	}
}
