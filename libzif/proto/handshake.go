package proto

import (
	"errors"

	"golang.org/x/crypto/ed25519"

	log "github.com/sirupsen/logrus"
	"github.com/zif/zif/libzif/common"
	"github.com/zif/zif/libzif/dht"
	"github.com/zif/zif/libzif/util"
)

// Perform a handshake operation given a peer. server.go does the other end of this.
func handshake(cl Client, lp common.Signer) (ed25519.PublicKey, error) {
	header, err := handshake_recieve(cl)

	if err != nil {
		cl.WriteMessage(Message{Header: ProtoNo, Content: []byte(err.Error())})
		return header, err
	}

	if lp == nil {
		cl.WriteMessage(Message{Header: ProtoNo, Content: []byte("nil LocalPeer")})
		return header, errors.New("Handshake passed nil LocalPeer")
	}

	cl.WriteMessage(Message{Header: ProtoOk})
	err = handshake_send(cl, lp)

	if err != nil {
		return header, err
	}

	return header, nil
}

// Just recieves a handshake from a peer.
func handshake_recieve(cl Client) (ed25519.PublicKey, error) {
	check := func(e error) bool {
		if e != nil {
			log.Error(e.Error())
			cl.Close()
			return true
		}

		return false
	}

	log.Debug("Receiving handshake")

	/*var short int16 = 0x0000
	binary.Read(cl.conn, binary.BigEndian, &short)

	if short != ProtoZif {
		return nil, errors.New("This is not a Zif connection")
	}

	log.Debug("Zif connection")

	binary.Read(cl.conn, binary.BigEndian, &short)

	if short != ProtoVersion {
		return nil, errors.New("Incorrect protocol version")
	}

	log.Debug("Correct version")*/

	header, err := cl.ReadMessage()
	log.Debug("Read header")

	if check(err) {
		cl.WriteMessage(Message{Header: ProtoNo})
		return nil, err
	}

	log.Debug("Header recieved")

	err = cl.WriteMessage(Message{Header: ProtoOk})

	if check(err) {
		return nil, err
	}

	address := dht.Address{}
	address.Generate(header.Content)

	log.WithFields(log.Fields{"peer": address.String()}).Info("Incoming connection")

	// Send the client a cookie for them to sign, this proves they have the
	// private key, and it is highly unlikely an attacker has a signed cookie
	// cached.
	cookie, err := util.CryptoRandBytes(20)

	if check(err) {
		return nil, err
	}

	err = cl.WriteMessage(Message{Header: ProtoCookie, Content: cookie})

	if check(err) {
		return nil, err
	}

	sig, err := cl.ReadMessage()

	if check(err) {
		return nil, err
	}

	verified := ed25519.Verify(header.Content, cookie, sig.Content)

	if !verified {
		log.Error("Failed to verify peer ", address.String())
		cl.WriteMessage(Message{Header: ProtoNo})
		cl.Close()
		return nil, errors.New("Signature not verified")
	}

	cl.WriteMessage(Message{Header: ProtoOk})

	log.WithFields(log.Fields{"peer": address.String()}).Info("Verified")

	return header.Content, nil
}

// Sends a handshake to a peer.
func handshake_send(cl Client, lp common.Signer) error {
	log.Debug("Handshaking with ", cl.conn.RemoteAddr().String())

	/*binary.Write(cl.conn, binary.BigEndian, ProtoZif)
	binary.Write(cl.conn, binary.BigEndian, ProtoVersion)*/

	header := Message{
		Header:  ProtoHeader,
		Content: lp.PublicKey(),
	}

	err := cl.WriteMessage(header)

	if err != nil {
		return err
	}

	msg, err := cl.ReadMessage()

	if err != nil {
		return err
	}

	if !msg.Ok() {
		return errors.New("Peer refused header")
	}

	log.Debug("Header sent")

	msg, err = cl.ReadMessage()
	log.Debug("Cookie")

	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("Cookie recieved, signing")

	sig := lp.Sign(msg.Content)

	msg = &Message{
		Header:  ProtoSig,
		Content: sig,
	}

	cl.WriteMessage(msg)

	msg, err = cl.ReadMessage()
	log.Debug("Written cookie")

	if err != nil {
		return err
	}

	if !msg.Ok() {
		return errors.New("Peer refused signature")
	}

	log.Info("Handshake sent ok")

	return nil
}
