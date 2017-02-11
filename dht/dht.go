package dht

type DHT struct {
	db *NetDB
}

func NewDHT(addr Address, path string) *DHT {
	ret := &DHT{}

	db, err := NewNetDB(addr, path)

	if err != nil {
		panic(err)
	}

	ret.db = db

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
	return dht.db.Query(addr)
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
