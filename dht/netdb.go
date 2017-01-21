package dht

import (
	"encoding/json"
	"io/ioutil"

	"github.com/peterbourgon/diskv"
	log "github.com/sirupsen/logrus"
)

const (
	BucketSize = 20
)

type NetDB struct {
	table    [][]Address
	addr     Address
	database *diskv.Diskv
}

func NewNetDB(addr Address, path string) *NetDB {
	ret := &NetDB{}
	ret.addr = addr

	// One bucket of addresses per bit in an address
	// At the time of writing, uses roughly 64KB of memory
	ret.table = make([][]Address, AddressBinarySize*8)

	// allocate each bucket
	for n, _ := range ret.table {
		ret.table[n] = make([]Address, 0, BucketSize)
	}

	// setup diskv
	transform := func(s string) []string {
		return []string{}
	}

	ret.database = diskv.New(diskv.Options{
		BasePath:     path,
		Transform:    transform,
		CacheSizeMax: 10 * 1024 * 1024,
	})

	return ret
}

func (ndb *NetDB) TableLen() int {
	size := 0

	for _, i := range ndb.table {
		size += len(i)
	}

	return size
}

func (ndb *NetDB) Insert(kv *KeyValue) error {
	if !kv.Valid() {
		s, _ := kv.Key().String()
		return &InvalidValue{s}
	}

	// Find the distance between the kv address and our own address, this is the
	// index in the table
	index := kv.Key().Xor(&ndb.addr).LeadingZeroes()
	bucket := ndb.table[index]

	// there is capacity, insert at the front
	// search to see if it is already inserted

	found := -1

	for n, i := range bucket {
		if i.Equals(kv.Key()) {
			found = n
			break
		}
	}

	// if it already exists, it first needs to be removed from it's old position
	if found != -1 {
		bucket = append(bucket[:found], bucket[found+1:]...)
	} else if len(bucket) == BucketSize {
		// TODO: Ping all peers, remove any inactive
		return &NoCapacity{BucketSize}
	}

	bucket = append([]Address{*kv.Key()}, bucket...)

	ndb.table[index] = bucket

	// key has been added to the routing table, now store the entry!
	s, _ := kv.Key().String()
	ndb.database.Write(s, kv.Value())

	return nil
}

// Returns the KeyValue if this node has the address, nil and err otherwise.
func (ndb *NetDB) Query(addr Address) (*KeyValue, error) {
	s, _ := addr.String()
	if !ndb.database.Has(s) {
		return nil, nil
	}

	value, err := ndb.database.Read(s)

	if err != nil {
		return nil, err
	}

	kv := NewKeyValue(addr, value)

	// reinsert the kv, popular things will stay near the top
	return kv, ndb.Insert(kv)
}

func (ndb *NetDB) Has(addr Address) bool {
	s, _ := addr.String()
	return ndb.database.Has(s)
}

func (ndb *NetDB) queryAddresses(as []Address) Pairs {
	ret := make(Pairs, 0, len(as))

	for _, i := range as {
		kv, err := ndb.Query(i)

		if err != nil {
			continue
		}

		ret = append(ret, kv)
	}

	return ret
}

func (ndb *NetDB) FindClosest(addr Address) (Pairs, error) {
	// Find the distance between the kv address and our own address, this is the
	// index in the table
	index := addr.Xor(&ndb.addr).LeadingZeroes()
	bucket := ndb.table[index]

	if len(bucket) == BucketSize {
		return ndb.queryAddresses(bucket), nil
	}

	ret := make(Pairs, 0, BucketSize)

	// Start with bucket, copy all across, then move left outwards checking all
	// other buckets.
	for i := 0; (index-i >= 0 || index+i <= len(addr.Raw)*8) &&
		len(ret) < BucketSize; i++ {

		if index-i >= 0 {
			bucket = ndb.table[index-i]

			for _, i := range bucket {
				if len(bucket) >= BucketSize {
					break
				}

				kv, err := ndb.Query(i)

				if err != nil {
					continue
				}

				ret = append(ret, kv)
			}
		}

		if index+i < len(addr.Raw)*8 {
			bucket = ndb.table[index+i]

			for _, i := range bucket {
				if len(bucket) >= BucketSize {
					break
				}

				kv, err := ndb.Query(i)

				if err != nil {
					continue
				}

				ret = append(ret, kv)
			}
		}

	}

	return ret, nil
}

func (ndb *NetDB) SaveTable(path string) {
	data, err := json.Marshal(ndb.table)

	if err != nil {
		log.Error(err.Error())
	}

	ioutil.WriteFile(path, data, 0644)

}

func (ndb *NetDB) LoadTable(path string) {
	raw, _ := ioutil.ReadFile(path)

	json.Unmarshal(raw, &ndb.table)
}
