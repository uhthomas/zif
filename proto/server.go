package proto

// tcp server

import (
	"encoding/binary"
	"io"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zif/zif/common"
	"github.com/zif/zif/util"
)

type Server struct {
	listener net.Listener
}

func (s *Server) Listen(addr string, handler ProtocolHandler, data common.Encodable) {
	var err error

	s.listener, err = net.Listen("tcp", addr)

	if err != nil {
		panic(err)
	}

	log.Info("Listening on ", addr)

	for {
		conn, err := s.listener.Accept()

		if err != nil {
			log.Error(err.Error())
		}

		log.Info("New TCP connection")

		var zif int16
		binary.Read(conn, binary.LittleEndian, &zif)

		if zif != ProtoZif {
			log.Error("This is not a Zif connection: ", zif)
			continue
		}

		log.Debug("Zif connection")

		var version int16
		binary.Read(conn, binary.LittleEndian, &version)

		if version != ProtoVersion {
			log.Error("Incorrect protocol version: ", version)
			continue
		}

		log.Debug("Correct version")

		log.Debug("Handshaking new connection")
		go s.Handshake(conn, handler, data)
	}
}

func (s *Server) ListenStream(peer NetworkPeer, handler ProtocolHandler) {
	// Allowed to open 4 streams per second, bursting to three.
	limiter := util.NewLimiter(time.Second/4, 3, true)
	defer limiter.Stop()

	session := peer.Session()

	for {
		stream, err := session.Accept()
		limiter.Wait()

		if err != nil {
			if err == io.EOF {
				log.Info("Peer closed connection")
				handler.HandleCloseConnection(peer.Address())

			} else {
				log.Error(err.Error())
			}

			return
		}

		log.Debug("Accepted stream (", session.NumStreams(), " total)")

		peer.AddStream(stream)

		go s.HandleStream(peer, handler, stream)
	}
}

func (s *Server) HandleStream(peer NetworkPeer, handler ProtocolHandler, stream net.Conn) {
	log.Debug("Handling stream")

	cl := Client{stream, nil, nil}

	for {
		msg, err := cl.ReadMessage()

		if err != nil {
			log.Error(err.Error())
			return
		}
		msg.Client = &cl
		msg.From = peer.Address()

		s.RouteMessage(msg, handler)
	}
}

func (s *Server) RouteMessage(msg *Message, handler ProtocolHandler) {
	var err error

	switch msg.Header {

	case ProtoDhtAnnounce:
		err = handler.HandleAnnounce(msg)
	case ProtoDhtQuery:
		err = handler.HandleQuery(msg)
	case ProtoDhtFindClosest:
		err = handler.HandleFindClosest(msg)
	case ProtoSearch:
		err = handler.HandleSearch(msg)
	case ProtoRecent:
		err = handler.HandleRecent(msg)
	case ProtoPopular:
		err = handler.HandlePopular(msg)
	case ProtoRequestHashList:
		err = handler.HandleHashList(msg)
	case ProtoRequestPiece:
		err = handler.HandlePiece(msg)
	case ProtoRequestAddPeer:
		err = handler.HandleAddPeer(msg)
	case ProtoPing:
		err = handler.HandlePing(msg)

	default:
		log.Error("Unknown message type")

	}

	if err != nil {
		log.Error(err.Error())
	}

}

func (s *Server) Handshake(conn net.Conn, lp ProtocolHandler, data common.Encodable) {
	cl := Client{conn, nil, nil}

	header, err := handshake(cl, lp, data)

	if err != nil {
		log.Error(err.Error())
		return
	}

	peer, err := lp.HandleHandshake(ConnHeader{cl, *header})

	if err != nil {
		log.Error(err.Error())
		return
	}

	go s.ListenStream(peer, lp)
}

func (s *Server) Close() {
	if s.listener != nil {
		s.listener.Close()
	}
}
