package dht

/*
	This file stores all the SQL queries needed for the NetDB.
	It will also be used to prepare all SQL statements :)
*/

const (
	/*
		id             - primary key
		address        - the raw Zif address, stored as a binary blob
		name           - the user-defined name of the node
		desc           - the user-defined description of the node
		publicAddress  - the user-defined address of the node on the internet/tor/etc
		port           - the port used to connect to the node
		publicKey      - the publicKey for the node, can be checked with the address
		signature      - this entry, signed
		collectionSig  - the signature of the hash of all of the posts
		collectionHash - the root hash of all the posts
		postCount      - the number of posts this node has
		seedCount      - the number of seeds this node has
		updated        - when this entry was last updated by the node, or another adding seeds
		seen           - when this node was last seen online
	*/
	sqlCreateEntriesTable = `
			CREATE TABLE IF NOT EXISTS
				entry(
					id INTEGER PRIMARY KEY NOT NULL,
					address BLOB(20) UNIQUE NOT NULL,
					name STRING NOT NULL,
					desc STRING,
					publicAddress STRING(253) NOT NULL,
					port INT,
					publicKey BLOB NOT NULL,
					signature BLOB,
					collectionSig BLOB,
					collectionHash BLOB,
					postCount INT,
					seedCount INT,
					seedingCount INT,
					updated INT,
					seen INT
				)
	`

	// Create the seeds table, using to link together seeds and the actual node
	sqlCreateSeedsTable = `
		CREATE TABLE IF NOT EXISTS 
				seed(
					id INTEGER PRIMARY KEY NOT NULL,
					seed INTEGER NOT NULL,
					for INTEGER NOT NULL
				)
	`
	// The full text search virtual table, allowing for the search of a node by
	// description and name.
	sqlCreateFtsTable = `
			CREATE VIRTUAL TABLE IF NOT EXISTS
				ftsEntry using fts4(
					content="entries",
					name,
					desc,
				)
	`
	sqlInsertEntry = `
			INSERT OR IGNORE INTO entry (
				address,
				name,
				desc,
				publicAddress,
				port,
				publicKey,
				signature,
				collectionSig,
				collectionHash,
				postCount,
				seedCount,
				seedingCount,
				updated,
				seen
			)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	sqlInsertFtsEntry = `
			INSERT OR IGNORE INTO ftsEntry (
				docid,
				name,
				desc
			) VALUES(?, ?, ?)
	`

	// We need an index on addresses, as nodes wll be fetched by index really
	// quite often. Most of the time actually! It's probably a good idea to cache
	// in RAM for n seconds.
	sqlIndexAddresses = `
			CREATE INDEX IF NOT EXISTS
				addressIndex ON entry(address)
	`

	sqlQueryAddress = `
		SELECT * FROM entry WHERE address=?
	`
)
