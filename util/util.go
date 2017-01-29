package util

import (
	"bufio"
	"crypto/rand"
	"io"
	"math/big"

	log "github.com/sirupsen/logrus"
)

func CryptoRandBytes(size int) ([]byte, error) {
	buf := make([]byte, size)
	_, err := rand.Read(buf)

	if err != nil {
		return nil, err
	}

	return buf, nil
}

func CryptoRandInt(min, max int64) int64 {
	num, err := rand.Int(rand.Reader, big.NewInt(max-min))

	if err != nil {
		log.Error(err.Error())
		return min
	}

	return num.Int64() + min
}

func ReadPost(r io.Reader, delim byte) {
	br := bufio.NewReader(r)

	br.ReadString(delim)
}

// Returns everything in two that is not in one
func SliceDiff(one [][]byte, two [][]byte) [][]byte {
	// first build a map out of one
	encountered := make(map[string]bool)
	result := make([][]byte, 0, len(one)+len(two))

	for _, i := range one {
		encountered[string(i)] = true
	}

	// then add to result if not in the map
	for _, i := range two {
		if _, ok := encountered[string(i)]; !ok {
			result = append(result, i)
		}
	}

	return result
}

func MergeSeeds(one [][]byte, two [][]byte) [][]byte {
	// make a map
	encountered := make(map[string]bool)
	result := make([][]byte, 0, len(one)+len(two))

	for _, i := range one {
		encountered[string(i)] = true
	}

	for _, i := range two {
		encountered[string(i)] = true
	}

	for k, _ := range encountered {
		result = append(result, []byte(k))
	}

	return result
}
