package data

import "testing"

const ArchInfoHash = "657c483dc66c1f248fc2eda5f5682ea557233e7a"
const UbuntuInfoHash = "9f9165d9a281a9b8e782cd5176bbcc8256fd1871"

func ConnectDb(t *testing.T) *zif.Database {
	db := zif.NewDatabase("file::memory:?cache=shared")

	err := db.Connect()

	if err != nil {
		t.Error(err.Error())
	}

	return db
}

func TestDatabaseInsert(t *testing.T) {
	db := ConnectDb(t)
	defer db.Close()

	source, _ := zif.CryptoRandBytes(20)
	post := zif.NewPost(ArchInfoHash, "Arch 2016-09-03", 100, 10, 1472860800, source)

	err := db.InsertPost(post)
	test_error(err, t)

	recent, err := db.QueryRecent(0)
	test_error(err, t)

	if len(recent) == 0 {
		t.Fatal("Database insert failed")
	}

	if recent[0].InfoHash != ArchInfoHash {
		t.Error("InfoHash did not match")
	}

	if db.PostCount() != 1 {
		t.Error("Database does not contain the correct number of posts")
	}
}

func test_error(err error, t *testing.T) {
	if err != nil {
		t.Error(err.Error())
	}
}

func TestDatabaseSearch(t *testing.T) {
	db := ConnectDb(t)
	source, _ := zif.CryptoRandBytes(20)

	arch := zif.NewPost(ArchInfoHash, "Arch Linux 2016-09-03", 100, 10, 1472860800, source)
	ubuntu := zif.NewPost(UbuntuInfoHash, "Ubuntu Linux 16.04.1", 101, 9, 1472860800, source)

	err := db.InsertPost(arch)
	test_error(err, t)

	err = db.InsertPost(ubuntu)
	test_error(err, t)

	err = db.GenerateFts(0)
	test_error(err, t)

	results, err := db.Search("arch", 0)
	test_error(err, t)

	if len(results) == 0 {
		t.Error("Search returned no results")
	}

	if results[0].InfoHash != ArchInfoHash {
		t.Error("Search not correctly performed")
	}

	results, err = db.Search("ubuntu", 0)
	test_error(err, t)

	if len(results) == 0 {
		t.Error("Search returned no results")
	}

	if results[0].InfoHash != UbuntuInfoHash {
		t.Error("Search not correctly performed")
	}

	results, err = db.Search("linux", 0)
	test_error(err, t)

	if len(results) != 2 {
		t.Error("Search did not return all results")
	}

	if results[0].InfoHash != UbuntuInfoHash || results[1].InfoHash != ArchInfoHash {
		t.Error("Results not correctly ordered")
	}

}
