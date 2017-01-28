package proto

import (
	"errors"

	"golang.org/x/crypto/ed25519"

	log "github.com/sirupsen/logrus"
	"github.com/zif/zif/common"
	"github.com/zif/zif/util"
)

// Perform a handshake operation given a peer. server.go does the other end of this.
func handshake(cl Client, lp common.Signer, data common.Encodable) (*Entry, error) {
	header, err := handshake_recieve(cl)

	if err != nil {
		cl.WriteErr(err)
		return header, err
	}

	if lp == nil {
		cl.WriteErr(errors.New("Nil localpeer"))
		return header, errors.New("Handshake passed nil LocalPeer")
	}

	cl.WriteMessage(Message{Header: ProtoOk})
	err = handshake_send(cl, lp, data)

	if err != nil {
		return header, err
	}

	return header, nil
}

// Just recieves a handshake from a peer.
func handshake_recieve(cl Client) (*Entry, error) {
	check := func(e error) bool {
		if e != nil {
			log.Error(e.Error())
			cl.Close()
			return true
		}

		return false
	}

	log.Debug("Receiving handshake")

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

	var entry Entry
	err = header.Read(&entry)

	if err != nil {
		return nil, err
	}

	err = entry.Verify()
	if err != nil {
		return nil, err
	}

	s, _ := entry.Address.String()
	log.WithFields(log.Fields{"peer": s}).Info("Incoming connection")

	// Send the client a cookie for them to sign, this proves they have the
	// private key, and it is highly unlikely an attacker has a signed cookie
	// cached.
	cookie, err := util.CryptoRandBytes(20)

	if check(err) {
		return nil, err
	}
	msg := Message{Header: ProtoCookie}
	err = msg.Write(cookie)

	if err != nil {
		return nil, err
	}

	err = cl.WriteMessage(msg)

	if check(err) {
		return nil, err
	}

	sig, err := cl.ReadMessage()

	if check(err) {
		return nil, err
	}

	// need to decompress the signature before verifying
	var signature [ed25519.SignatureSize]byte
	sig.Read(&signature)
	verified := ed25519.Verify(entry.PublicKey, cookie, signature[:])

	if !verified {
		log.Error("Failed to verify peer ", s)
		cl.WriteMessage(Message{Header: ProtoNo})
		cl.Close()
		return nil, errors.New("Signature not verified")
	}

	cl.WriteMessage(Message{Header: ProtoOk})

	log.WithFields(log.Fields{"peer": s}).Info("Verified")

	return &entry, nil
}

// Sends a handshake to a peer.
func handshake_send(cl Client, lp common.Signer, data common.Encodable) error {
	log.Debug("Handshaking with ", cl.conn.RemoteAddr().String())

	/*binary.Write(cl.conn, binary.BigEndian, ProtoZif)
	binary.Write(cl.conn, binary.BigEndian, ProtoVersion)*/

	header := Message{
		Header: ProtoHeader,
	}

	err := header.Write(data)

	if err != nil {
		return err
	}

	err = cl.WriteMessage(header)

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

	// the peer expects us to sign the *decompressed* cookie. So do that.
	var cookie [20]byte
	msg.Read(&cookie)
	sig := lp.Sign(cookie[:])

	msg = &Message{
		Header: ProtoSig,
	}

	err = msg.Write(sig)

	if err != nil {
		return err
	}

	err = cl.WriteMessage(msg)

	if err != nil {
		return err
	}

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
