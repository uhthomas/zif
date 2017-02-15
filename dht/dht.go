package dht

import (
	"database/sql"
	log "github.com/sirupsen/logrus"
)

type DHT struct {
	db *NetDB
}

// sets up the dht
func NewDHT(addr Address, path string) *DHT {
	ret := &DHT{}

	db, err := NewNetDB(addr, path)

	if err != nil {
		panic(err)
	}

	ret.db = db

	log.Debug("Loading latest into DHT")
	// insert a load of new entries, keep it fresh!
	entries, err := db.QueryLatest()

	if err == sql.ErrNoRows {
		return ret
	}

	count := 0
	for _, i := range entries {
		count += 1
		db.Insert(i)
	}

	log.WithField("count", count).Debug("Inserted")

	return ret
}

func (dht *DHT) Address() Address {
	return dht.db.addr
}

func (dht *DHT) Insert(entry Entry) error {
	// TODO: Announces
	return dht.db.Insert(entry)
}

func (dht *DHT) Query(addr Address) (*Entry, error) {
	entry, _, err := dht.db.Query(addr)

	return entry, err
}

func (dht *DHT) FindClosest(addr Address) (Entries, error) {
	return dht.db.FindClosest(addr)
}

func (dht *DHT) SaveTable(path string) {
	dht.db.SaveTable(path)
}

func (dht *DHT) LoadTable(path string) {
	dht.db.LoadTable(path)
}
