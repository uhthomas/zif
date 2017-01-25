package util

import (
	"bufio"
	"crypto/rand"
	"io"
	"math/big"
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
		panic(err)
	}

	return num.Int64() + min
}

func ReadPost(r io.Reader, delim byte) {
	br := bufio.NewReader(r)

	br.ReadString(delim)
}

func MergeSeeds(one [][]byte, two [][]byte) [][]byte {
	temp := append(one, two...)

	// make a map
	encountered := make(map[string]bool)
	result := make([][]byte, 0, len(one)+len(two))

	for _, i := range temp {
		encountered[string(i)] = true
	}

	for k, _ := range encountered {
		result = append(result, []byte(k))
	}

	return result
}
