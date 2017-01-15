// Used to control the Zif daemon

package libzif

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"
)

type HttpServer struct {
	CommandServer *CommandServer
}

func (hs *HttpServer) ListenHttp(addr string) {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/", hs.IndexHandler)

	// This should be the ONLY route where the address is a non-Zif address

	router.HandleFunc("/peer/{address}/ping/", hs.Ping)
	router.HandleFunc("/peer/{address}/announce/", hs.Announce)
	router.HandleFunc("/peer/{address}/rsearch/", hs.PeerRSearch).Methods("POST")
	router.HandleFunc("/peer/{address}/search/", hs.PeerSearch).Methods("POST")
	router.HandleFunc("/peer/{address}/recent/{page}/", hs.Recent)
	router.HandleFunc("/peer/{address}/popular/{page}/", hs.Popular)
	router.HandleFunc("/peer/{address}/mirror/", hs.Mirror)
	router.HandleFunc("/peer/{address}/mirrorprogress/", hs.MirrorProgress)
	router.HandleFunc("/peer/{address}/index/{since}/", hs.PeerFtsIndex)

	router.HandleFunc("/self/addpost/", hs.AddPost).Methods("POST")
	router.HandleFunc("/self/index/{since}/", hs.FtsIndex)
	router.HandleFunc("/self/resolve/{address}/", hs.Resolve)
	router.HandleFunc("/self/bootstrap/{address}/", hs.Bootstrap)
	router.HandleFunc("/self/search/", hs.SelfSearch).Methods("POST")
	router.HandleFunc("/self/suggest/", hs.SelfSuggest).Methods("POST")
	router.HandleFunc("/self/recent/{page}/", hs.SelfRecent)
	router.HandleFunc("/self/popular/{page}/", hs.SelfPopular)
	router.HandleFunc("/self/addmeta/{pid}/", hs.AddMeta).Methods("POST")
	router.HandleFunc("/self/savecollection/", hs.SaveCollection)
	router.HandleFunc("/self/rebuildcollection/", hs.RebuildCollection)
	router.HandleFunc("/self/peers/", hs.Peers)
	router.HandleFunc("/self/requestaddpeer/{remote}/{peer}/", hs.RequestAddPeer)
	router.HandleFunc("/self/set/{key}/", hs.SelfSet).Methods("POST")
	router.HandleFunc("/self/get/{key}/", hs.SelfGet)

	log.Info("Starting HTTP server on ", addr)

	err := http.ListenAndServe(addr, router)

	if err != nil {
		panic(err)
	}
}

func write_http_response(w http.ResponseWriter, cr CommandResult) {
	var err int

	if cr.IsOK {
		err = http.StatusOK
	} else {
		err = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(err)

	cr.WriteJSON(w)
}

func (hs *HttpServer) Ping(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	write_http_response(w, hs.CommandServer.Ping(CommandPing{vars["address"]}))
}
func (hs *HttpServer) Announce(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	write_http_response(w, hs.CommandServer.Announce(CommandAnnounce{vars["address"]}))
}
func (hs *HttpServer) PeerRSearch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	addr := vars["address"]

	query := r.FormValue("query")
	page := r.FormValue("page")

	pagei, err := strconv.Atoi(page)
	if err != nil {
		write_http_response(w, CommandResult{false, nil, err})
		return
	}

	write_http_response(w, hs.CommandServer.RSearch(
		CommandRSearch{CommandPeer{addr}, query, pagei}))
}
func (hs *HttpServer) PeerSearch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	addr := vars["address"]

	query := r.FormValue("query")
	page := r.FormValue("page")

	pagei, err := strconv.Atoi(page)
	if err != nil {
		write_http_response(w, CommandResult{false, nil, err})
		return
	}

	write_http_response(w, hs.CommandServer.PeerSearch(
		CommandPeerSearch{CommandPeer{addr}, query, pagei}))
}
func (hs *HttpServer) Recent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	addr := vars["address"]
	page := vars["page"]

	pagei, err := strconv.Atoi(page)
	if err != nil {
		write_http_response(w, CommandResult{false, nil, err})
		return
	}

	write_http_response(w, hs.CommandServer.PeerRecent(
		CommandPeerRecent{CommandPeer{addr}, pagei}))
}
func (hs *HttpServer) Popular(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	addr := vars["address"]
	page := vars["page"]

	pagei, err := strconv.Atoi(page)
	if err != nil {
		write_http_response(w, CommandResult{false, nil, err})
		return
	}

	write_http_response(w, hs.CommandServer.PeerPopular(
		CommandPeerPopular{CommandPeer{addr}, pagei}))
}
func (hs *HttpServer) Mirror(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	write_http_response(w, hs.CommandServer.Mirror(CommandMirror{vars["address"]}))
}

func (hs *HttpServer) MirrorProgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	write_http_response(w, hs.CommandServer.GetMirrorProgress(CommandMirrorProgress{vars["address"]}))
}

func (hs *HttpServer) PeerFtsIndex(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	addr := vars["address"]
	since := vars["since"]

	sincei, err := strconv.Atoi(since)
	if err != nil {
		write_http_response(w, CommandResult{false, nil, err})
		return
	}

	write_http_response(w, hs.CommandServer.PeerIndex(
		CommandPeerIndex{CommandPeer{addr}, sincei}))
}

func (hs *HttpServer) AddPost(w http.ResponseWriter, r *http.Request) {
	pj := r.FormValue("data")
	index := r.FormValue("index") == "true"

	var post CommandAddPost
	err := json.Unmarshal([]byte(pj), &post)

	if err != nil {
		write_http_response(w, CommandResult{false, nil, err})
		return
	}

	post.Index = index

	write_http_response(w, hs.CommandServer.AddPost(post))
}
func (hs *HttpServer) FtsIndex(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	since, err := strconv.Atoi(vars["since"])
	if err != nil {
		write_http_response(w, CommandResult{false, nil, err})
		return
	}

	write_http_response(w, hs.CommandServer.SelfIndex(
		CommandSelfIndex{since}))
}
func (hs *HttpServer) Resolve(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	write_http_response(w, hs.CommandServer.Resolve(CommandResolve{vars["address"]}))
}
func (hs *HttpServer) Bootstrap(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	write_http_response(w, hs.CommandServer.Bootstrap(CommandBootstrap{vars["address"]}))
}
func (hs *HttpServer) SelfSearch(w http.ResponseWriter, r *http.Request) {
	query := r.FormValue("query")
	page := r.FormValue("page")

	pagei, err := strconv.Atoi(page)
	if err != nil {
		write_http_response(w, CommandResult{false, nil, err})
		return
	}

	write_http_response(w, hs.CommandServer.SelfSearch(CommandSelfSearch{CommandSuggest{query}, pagei}))
}

func (hs *HttpServer) SelfSuggest(w http.ResponseWriter, r *http.Request) {
	log.Info("HTTP: Self Suggest request")

	query := r.FormValue("query")

	write_http_response(w, hs.CommandServer.SelfSuggest(CommandSuggest{query}))
}

// TODO: SelfSuggest after merge
func (hs *HttpServer) SelfRecent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	page, err := strconv.Atoi(vars["page"])
	if err != nil {
		write_http_response(w, CommandResult{false, nil, err})
		return
	}

	write_http_response(w, hs.CommandServer.SelfRecent(CommandSelfRecent{page}))
}
func (hs *HttpServer) SelfPopular(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	page, err := strconv.Atoi(vars["page"])
	if err != nil {
		write_http_response(w, CommandResult{false, nil, err})
		return
	}

	write_http_response(w, hs.CommandServer.SelfPopular(CommandSelfPopular{page}))
}
func (hs *HttpServer) AddMeta(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	pid, err := strconv.Atoi(vars["pid"])
	meta := r.FormValue("meta")

	if err != nil {
		write_http_response(w, CommandResult{false, nil, err})
		return
	}

	write_http_response(w, hs.CommandServer.AddMeta(
		CommandAddMeta{CommandMeta{pid}, meta}))
}
func (hs *HttpServer) SaveCollection(w http.ResponseWriter, r *http.Request) {
	write_http_response(w, hs.CommandServer.SaveCollection(nil))
}
func (hs *HttpServer) RebuildCollection(w http.ResponseWriter, r *http.Request) {
	write_http_response(w, hs.CommandServer.RebuildCollection(nil))
}
func (hs *HttpServer) Peers(w http.ResponseWriter, r *http.Request) {
	write_http_response(w, hs.CommandServer.Peers(nil))
}

func (hs *HttpServer) RequestAddPeer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	write_http_response(w, hs.CommandServer.RequestAddPeer(CommandRequestAddPeer{
		vars["remote"], vars["peer"],
	}))
}

func (hs *HttpServer) SelfSet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	key := vars["key"]
	value := r.FormValue("value")

	write_http_response(w, hs.CommandServer.LocalSet(CommandLocalSet{key, value}))
}

func (hs *HttpServer) SelfGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	key := vars["key"]

	write_http_response(w, hs.CommandServer.LocalGet(CommandLocalGet{key}))
}

func (hs *HttpServer) IndexHandler(w http.ResponseWriter, r *http.Request) {
	// TODO
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Zif"))
}
