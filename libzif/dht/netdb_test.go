package dht_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/zif/zif/libzif/dht"
	"github.com/zif/zif/libzif/util"
)

// would be so cool if you had that public key for real :O
var addr = dht.NewAddress([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
	15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
	27, 28, 29, 30, 31, 32})

var addr2 = dht.NewAddress([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
	15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
	27, 28, 29, 30, 31, 33})

func insert(t *testing.T, db *dht.NetDB, a dht.Address, el int) {
	err := db.Insert(dht.NewKeyValue(a, a.Raw))

	if err != nil {
		t.Error(err.Error())
	}

	if db.TableLen() != el {
		t.Errorf("TableLen not correct: %d, expected: %d", db.TableLen(), el)
	}
}

func newDB() (*dht.NetDB, func()) {
	dir, _ := ioutil.TempDir("", "zif")

	db := dht.NewNetDB(addr, dir)

	return db, func() { os.RemoveAll(dir) }
}

func TestNetDBInsert(t *testing.T) {
	db, cl := newDB()
	defer cl()

	fmt.Println(addr.Xor(&addr2).LeadingZeroes())

	// two inserts should result in just one
	insert(t, db, addr, 1)
	insert(t, db, addr, 1)

	insert(t, db, addr2, 2)
}

func BenchmarkNetDBInsert(b *testing.B) {
	db, cl := newDB()
	defer cl()

	for i := 0; i < b.N; i++ {
		data, _ := util.CryptoRandBytes(20)
		db.Insert(dht.NewKeyValue(dht.Address{data}, data))
	}
}

func TestNetDBQuery(t *testing.T) {
	db, cl := newDB()
	defer cl()

	insert(t, db, addr, 1)
	insert(t, db, addr2, 2)

	kv, err := db.Query(addr)

	if err != nil {
		t.Error(err.Error())
	}

	if !kv.Key.Equals(&addr) || !bytes.Equal(kv.Value, addr.Raw) {
		t.Error("Query returned invalid data")
	}

	dat, _ := util.CryptoRandBytes(20)
	randAddr := dht.Address{dat}

	kv, err = db.Query(randAddr)

	if err == nil {
		t.Error("Random query did not error as expected")
	}
}

func BenchmarkNetDBQuery(b *testing.B) {
	db, cl := newDB()
	defer cl()

	db.Insert(dht.NewKeyValue(addr, addr.Raw))

	for i := 0; i < b.N; i++ {
		db.Query(addr)
	}
}

func TestNetDBFindClosest(t *testing.T) {
	db, cl := newDB()
	defer cl()

	insert(t, db, addr, 1)
	insert(t, db, addr2, 2)

	pairs, err := db.FindClosest(addr)

	if err != nil {
		t.Error(err.Error())
	}

	if len(pairs) != 1 {
		t.Error(fmt.Sprintf("Incorrect length returned: %d", len(pairs)))
	}

	if !pairs[0].Key.Equals(&addr2) {
		t.Error("Incorrect address returned")
	}
}

func BenchmarkNetDBFindClosest(b *testing.B) {
	db, cl := newDB()
	defer cl()

	for i := 0; i < dht.BucketSize*160; i++ {
		dat, _ := util.CryptoRandBytes(20)
		addr := dht.Address{dat}

		db.Insert(dht.NewKeyValue(addr, addr.Raw))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dat, _ := util.CryptoRandBytes(20)
		addr := dht.Address{dat}

		db.FindClosest(addr)
	}
}
