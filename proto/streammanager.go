// Keeps track of open TCP connections, as well as yamux sessions

package proto

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/proxy"

	"github.com/hashicorp/yamux"
	log "github.com/sirupsen/logrus"
	"github.com/zif/zif/common"
)

type StreamManager struct {
	connection ConnHeader

	// Open yamux servers
	server *yamux.Session

	// Open yamux clients
	client *yamux.Session

	// Open yamux streams
	clients []Client

	Socks     bool
	SocksPort int
	torDialer proxy.Dialer
}

func (sm *StreamManager) SetConnection(conn ConnHeader) {
	sm.connection = conn
}

func (sm *StreamManager) Setup() {
	sm.server = nil
	sm.client = nil
	sm.clients = make([]Client, 0, 10)
}

func (sm *StreamManager) OpenSocks(addr string, lp ProtocolHandler, data common.Encodable) (*ConnHeader, error) {
	if sm.torDialer == nil {
		dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", sm.SocksPort), nil, proxy.Direct)

		if err != nil {
			return nil, err
		}

		sm.torDialer = dialer
	}

	conn, err := sm.torDialer.Dial("tcp", addr)

	if err != nil {
		return nil, err
	}

	return sm.handleConnection(conn, lp, data)
}

func (sm *StreamManager) OpenTCP(addr string, lp ProtocolHandler, data common.Encodable) (*ConnHeader, error) {
	if sm.Socks {
		return sm.OpenSocks(addr, lp, data)
	}

	if sm.connection.Client.conn != nil {
		return &sm.connection, nil
	}

	conn, err := net.Dial("tcp", addr)

	if err != nil {
		return nil, err
	}

	return sm.handleConnection(conn, lp, data)
}

func (sm *StreamManager) handleConnection(conn net.Conn, lp ProtocolHandler, data common.Encodable) (*ConnHeader, error) {
	log.WithField("zif", ProtoZif).Info("Sending")
	err := binary.Write(conn, binary.LittleEndian, ProtoZif)

	if err != nil {
		return nil, err
	}

	log.WithField("version", ProtoVersion).Info("Sending")
	err = binary.Write(conn, binary.LittleEndian, ProtoVersion)

	if err != nil {
		return nil, err
	}

	header, err := sm.Handshake(conn, lp, data)

	if err != nil {
		return nil, err
	}

	if header == nil {
		return nil, errors.New("Failed to handshake, nil entry")
	}

	pair := ConnHeader{*NewClient(conn), *header}
	sm.connection = pair

	return &pair, nil
}

func (sm *StreamManager) Handshake(conn net.Conn, lp ProtocolHandler, data common.Encodable) (*Entry, error) {
	cl := NewClient(conn)
	log.Debug("Sending handshake")
	err := handshake_send(*cl, lp, data)

	msg, err := cl.ReadMessage()

	if err != nil {
		return nil, nil
	}

	if !msg.Ok() {
		return nil, errors.New(string(msg.Content))
	}

	// server now knows that we are definitely who we say we are.
	// but...
	// is the server who we think it is?
	// better check!
	server_header, err := handshake_recieve(*cl)

	if err != nil {
		return server_header, err
	}

	log.Info("Handshake complete")

	return server_header, nil
}

func (sm *StreamManager) ConnectClient() (*yamux.Session, error) {
	// If there is already a client connected, return that.
	if sm.client != nil {
		return sm.client, nil
	}

	if sm.server != nil {
		return nil, errors.New("There is already a server connected to that socket")
	}

	client, err := yamux.Client(sm.connection.Client.conn, nil)

	if err != nil {
		return nil, err
	}

	sm.client = client

	return client, nil
}

func (sm *StreamManager) ConnectServer() (*yamux.Session, error) {
	// If there is already a server connected, return that.
	if sm.server != nil {
		return sm.server, nil
	}

	if sm.client != nil {
		return nil, errors.New("There is already a client connected to that socket")
	}

	server, err := yamux.Server(sm.connection.Client.conn, nil)

	if err != nil {
		return nil, err
	}

	sm.server = server

	return server, nil
}

func (sm *StreamManager) Close() {
	session := sm.GetSession()

	if session != nil {
		session.Close()
	}

	if sm.connection.Client.conn != nil {
		sm.connection.Client.Close()
	}
}

func (sm *StreamManager) GetSession() *yamux.Session {
	if sm.server != nil {
		return sm.server
	}

	if sm.client != nil {
		return sm.client
	}

	return nil
}

func (sm *StreamManager) OpenStream() (*Client, error) {
	var ret Client
	var err error
	session := sm.GetSession()

	if session == nil {
		return nil, errors.New("Cannot open stream, no session")
	}

	ret.conn, err = session.Open()

	if err != nil {
		return nil, err
	}

	err = ret.conn.SetDeadline(time.Now().Add(time.Second * 10))

	if err != nil {
		return nil, err
	}

	log.WithField("total", session.NumStreams()).Debug("Opened stream")
	return &ret, nil
}

// These streams should be coming from Server.ListenStream, as they will be started
// by the peer.
func (sm *StreamManager) AddStream(conn net.Conn) {
	var ret Client
	ret.conn = conn
	sm.clients = append(sm.clients, ret)
}

func (sm *StreamManager) GetStream(conn net.Conn) *Client {
	id := conn.(*yamux.Stream).StreamID()

	for _, c := range sm.clients {
		if c.conn.(*yamux.Stream).StreamID() == id {
			return &c
		}
	}

	return nil
}

func (sm *StreamManager) RemoveStream(conn net.Conn) {
	id := conn.(*yamux.Stream).StreamID()

	for i, c := range sm.clients {
		if c.conn.(*yamux.Stream).StreamID() == id {
			sm.clients = append(sm.clients[:i], sm.clients[i+1:]...)
		}
	}
}
