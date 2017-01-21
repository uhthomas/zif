package dht

type DHT struct {
	db *NetDB
}

func NewDHT(addr Address, path string) *DHT {
	ret := &DHT{
		db: NewNetDB(addr, path),
	}

	return ret
}

func (dht *DHT) Address() Address {
	return dht.db.addr
}

func (dht *DHT) Insert(kv *KeyValue) error {
	// TODO: Announces
	return dht.db.Insert(kv)
}

func (dht *DHT) Query(addr Address) (*KeyValue, error) {
	return dht.db.Query(addr)
}

func (dht *DHT) FindClosest(addr Address) (Pairs, error) {
	return dht.db.FindClosest(addr)
}

func (dht *DHT) SaveTable(path string) {
	dht.db.SaveTable(path)
}

func (dht *DHT) LoadTable(path string) {
	dht.db.LoadTable(path)
}

func (dht *DHT) Has(addr Address) bool {
	return dht.db.Has(addr)
}
