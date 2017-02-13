package dht

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

const (
	BucketSize = 20
)

type NetDB struct {
	table [][]Address
	addr  Address
	conn  *sql.DB

	stmtInsertEntry      *sql.Stmt
	stmtInsertFtsEntry   *sql.Stmt
	stmtEntryLen         *sql.Stmt
	stmtQueryAddress     *sql.Stmt
	stmtInsertSeed       *sql.Stmt
	stmtQueryIdByAddress *sql.Stmt
	stmtUpdateEntry      *sql.Stmt
}

func NewNetDB(addr Address, path string) (*NetDB, error) {
	var err error

	ret := &NetDB{}
	ret.addr = addr

	// One bucket of addresses per bit in an address
	// At the time of writing, uses roughly 64KB of memory
	ret.table = make([][]Address, AddressBinarySize*8)

	// allocate each bucket
	for n, _ := range ret.table {
		ret.table[n] = make([]Address, 0, BucketSize)
	}

	ret.conn, err = sql.Open("sqlite3", path)

	if err != nil {
		return nil, err
	}

	// don't bother preparing these, they are only used at startup

	// create the entries table first, it is most important
	_, err = ret.conn.Exec(sqlCreateEntriesTable)
	if err != nil {
		return nil, err
	}

	// store seed lists
	_, err = ret.conn.Exec(sqlCreateSeedsTable)
	if err != nil {
		return nil, err
	}

	// full text search
	_, err = ret.conn.Exec(sqlCreateFtsTable)
	if err != nil {
		return nil, err
	}

	// speed up entry lookups
	_, err = ret.conn.Exec(sqlIndexAddresses)
	if err != nil {
		return nil, err
	}

	// prepare all the SQL we will be needing
	ret.stmtInsertEntry, err = ret.conn.Prepare(sqlInsertEntry)
	if err != nil {
		return nil, err
	}

	ret.stmtInsertFtsEntry, err = ret.conn.Prepare(sqlInsertFtsEntry)
	if err != nil {
		return nil, err
	}

	ret.stmtInsertFtsEntry, err = ret.conn.Prepare("SELECT MAX(id) FROM entry")
	if err != nil {
		return nil, err
	}

	ret.stmtQueryAddress, err = ret.conn.Prepare(sqlQueryAddress)
	if err != nil {
		return nil, err
	}

	ret.stmtInsertSeed, err = ret.conn.Prepare(sqlInsertSeed)
	if err != nil {
		return nil, err
	}

	ret.stmtQueryIdByAddress, err = ret.conn.Prepare(sqlQueryIdByAddress)
	if err != nil {
		return nil, err
	}

	ret.stmtUpdateEntry, err = ret.conn.Prepare(sqlUpdateEntry)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// Get the total size of the in-memory routing table
func (ndb *NetDB) TableLen() int {
	size := 0

	for _, i := range ndb.table {
		size += len(i)
	}

	return size
}

// Get the total number of entries we have stored
func (ndb *NetDB) Len() (int, error) {
	var length int

	row := ndb.stmtEntryLen.QueryRow()
	err := row.Scan(&length)

	if err != nil {
		return -1, err
	}

	return length, err
}

// Insert an address into the in memory routing table. Theere is no need to store
// any data along with it as this can be fetched from the DB.
func (ndb *NetDB) insertIntoTable(addr Address) {
	// Find the distance between the kv address and our own address, this is the
	// index in the table
	index := addr.Xor(&ndb.addr).LeadingZeroes()
	bucket := ndb.table[index]

	// there is capacity, insert at the front
	// search to see if it is already inserted

	found := -1

	for n, i := range bucket {
		if i.Equals(&addr) {
			found = n
			break
		}
	}

	// if it already exists, it first needs to be removed from it's old position
	if found != -1 {
		bucket = append(bucket[:found], bucket[found+1:]...)
	} else if len(bucket) == BucketSize {

		// remove the back of the bucket, this update will go at the front
		bucket = bucket[:len(bucket)-1]
	}

	bucket = append([]Address{addr}, bucket...)

	ndb.table[index] = bucket
}

func (ndb *NetDB) insertIntoDB(entry Entry) error {

	addressString, err := entry.Address.String()

	if err != nil {
		return err
	}

	// Insert the entry into the main entry table
	_, err = ndb.stmtInsertEntry.Exec(addressString, entry.Name, entry.Desc,
		entry.PublicAddress, entry.Port, entry.PublicKey,
		entry.Signature, entry.CollectionSig, entry.CollectionHash,
		entry.PostCount, len(entry.Seeds), len(entry.Seeding),
		entry.Updated, entry.Seen)

	if err != nil {
		return err
	}

	// if that is all ok, then we can register all the seeds in the seed table
	// fun thing about this table, it can be used to populate both Seeds and
	// Seeding :D
	// Also need to make sure to not insert duplicates. SQL constraints should
	// do that for me. Woop woop!

	// first, register all the seeds for peers we are a seed for
	for _, i := range entry.Seeding {
		peer := Address{i}

		// we are a seed for this peer
		err := ndb.InsertSeed(peer, entry.Address)

		if err != nil {
			return err
		}
	}

	// then register all of the seeds for the current peer!
	for _, i := range entry.Seeds {
		peer := Address{i}

		// the peer is a seed for us
		err := ndb.InsertSeed(entry.Address, peer)

		if err != nil {
			return err
		}
	}

	return err
}

func (ndb *NetDB) InsertSeed(entry Address, seed Address) error {
	// First we need to map the addresses, which are essentially a network-wide
	// id, to an integer id which is local to our database.
	entryIdRes := ndb.stmtQueryIdByAddress.QueryRow(entry.Raw)
	seedIdRes := ndb.stmtQueryIdByAddress.QueryRow(seed.Raw)

	entryId := -1
	seedId := -1

	err := entryIdRes.Scan(&entryId)
	if err != nil {
		return err
	}

	err = seedIdRes.Scan(&seedId)
	if err != nil {
		return err
	}

	// got the ids, so now insert them into the database!
	_, err = ndb.stmtInsertSeed.Exec(seed, entry)

	return err
}

// Inserts an entry into both the routing table and the database
func (ndb *NetDB) Insert(entry Entry) error {
	err := entry.Verify()

	if err != nil {
		return err
	}

	ndb.insertIntoTable(entry.Address)

	// first we attempt to update the entry. If this succeeds, don't bother with
	// an insert :)

	err = ndb.Update(entry)

	if err != nil {
		err = ndb.insertIntoDB(entry)
	}

	return err
}

func (ndb *NetDB) Update(entry Entry) error {
	err := entry.Verify()

	if err != nil {
		return err
	}

	_, err = ndb.stmtUpdateEntry.Exec(entry.Name, entry.Desc, entry.PublicAddress,
		entry.Port, entry.PublicKey, entry.Signature, entry.CollectionSig,
		entry.CollectionHash, entry.PostCount, len(entry.Seeds), len(entry.Seeding),
		entry.Updated, entry.Seen)

	return err
}

// Returns the KeyValue if this node has the address, nil and err otherwise.
func (ndb *NetDB) Query(addr Address) (*Entry, error) {
	addressString, err := addr.String()

	if err != nil {
		return nil, err
	}

	ret := Entry{}
	row := ndb.stmtQueryAddress.QueryRow(addressString)

	id := 0
	seedCount := 0
	seedingCount := 0
	address := ""

	err = row.Scan(&id, &address, &ret.Name, &ret.Desc, &ret.PublicAddress,
		&ret.Port, &ret.PublicKey, &ret.Signature, &ret.CollectionSig, &ret.CollectionHash,
		&ret.PostCount, &seedCount, &seedingCount, &ret.Updated, &ret.Seen)

	if err != nil {
		return nil, err
	}

	decoded, err := DecodeAddress(address)

	if err != nil {
		return nil, err
	}

	ret.Seeding = make([][]byte, seedingCount)
	ret.Seeds = make([][]byte, seedCount)

	ret.Address.Raw = make([]byte, len(decoded.Raw))
	copy(ret.Address.Raw, decoded.Raw)

	// resinsert into the table, this keeps popular things easy to access
	// TODO: Store some sort of "lastQueried" in the database, then we have
	// even more data on how popular something is.
	// TODO: Make sure I'm not storing too much in the database :P
	ndb.insertIntoTable(ret.Address)
	return &ret, nil
}

func (ndb *NetDB) queryAddresses(as []Address) Entries {
	ret := make(Entries, 0, len(as))

	for _, i := range as {
		kv, err := ndb.Query(i)

		if err != nil {
			continue
		}

		ret = append(ret, kv)
	}

	return ret
}

func (ndb *NetDB) FindClosest(addr Address) (Entries, error) {
	// Find the distance between the kv address and our own address, this is the
	// index in the table
	index := addr.Xor(&ndb.addr).LeadingZeroes()
	bucket := ndb.table[index]

	if len(bucket) == BucketSize {
		return ndb.queryAddresses(bucket), nil
	}

	ret := make(Entries, 0, BucketSize)

	// Start with bucket, copy all across, then move left outwards checking all
	// other buckets.
	for i := 0; (index-i >= 0 || index+i <= len(addr.Raw)*8) &&
		len(ret) < BucketSize; i++ {

		if index-i >= 0 {
			bucket = ndb.table[index-i]

			for _, i := range bucket {
				if len(ret) >= BucketSize {
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
				if len(ret) >= BucketSize {
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
