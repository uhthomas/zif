package data

import (
	"bufio"
	"io"

	log "github.com/sirupsen/logrus"
)

type ErrorReader struct {
	reader *bufio.Reader
	Err    error
}

func NewErrorReader(r io.Reader) *ErrorReader {
	return &ErrorReader{bufio.NewReader(r), nil}
}

func (er *ErrorReader) ReadString(delim byte) string {
	var ret string

	ret, er.Err = er.reader.ReadString(delim)

	if er.Err != nil {
		log.Error(er.Err.Error())
		return ""
	}

	return ret[0 : len(ret)-1]
}

func (er *ErrorReader) ReadByte() byte {
	var ret byte

	ret, er.Err = er.reader.ReadByte()

	if er.Err != nil {
		return 0
	}

	return ret
}

type AddressResolutionError struct {
	Address string
}

func (a AddressResolutionError) Error() string {
	return "Failed to resolve address, address may not exist or is not reachable"
}
