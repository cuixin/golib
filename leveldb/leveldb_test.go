package leveldb

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"
)

var testConfigJson = []byte(`
    {
        "path" : "./testdb",
        "compression":true,
        "block_size" : 32768,
        "write_buffer_size" : 2097152,
        "cache_size" : 20971520
    }
    `)

var testOnce sync.Once
var testDB *DB

func getTestDB() *DB {
	f := func() {
		var err error
		testDB, err = Open(testConfigJson)
		if err != nil {
			println(err.Error())
			panic(err)
		}
	}

	testOnce.Do(f)
	return testDB
}

func TestSimple(t *testing.T) {
	db := getTestDB()

	key := []byte("key")
	value := []byte("hello world")
	if err := db.Put(key, value); err != nil {
		t.Fatal(err)
	}

	if v, err := db.Get(key); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(v, value) {
		t.Fatal("not equal")
	}

	if err := db.Delete(key); err != nil {
		t.Fatal(err)
	}

	if v, err := db.Get(key); err != nil {
		t.Fatal(err)
	} else if v != nil {
		t.Fatal("must nil")
	}
}

func TestBatch(t *testing.T) {
	db := getTestDB()

	key1 := []byte("key1")
	key2 := []byte("key2")

	value := []byte("hello world")

	db.Put(key1, value)
	db.Put(key2, value)

	wb := db.NewWriteBatch()
	defer wb.Close()

	wb.Delete(key2)
	wb.Put(key1, []byte("hello world2"))

	if err := wb.Commit(); err != nil {
		t.Fatal(err)
	}

	if v, err := db.Get(key2); err != nil {
		t.Fatal(err)
	} else if v != nil {
		t.Fatal("must nil")
	}

	if v, err := db.Get(key1); err != nil {
		t.Fatal(err)
	} else if string(v) != "hello world2" {
		t.Fatal(string(v))
	}

	wb.Delete(key1)

	wb.Rollback()

	if v, err := db.Get(key1); err != nil {
		t.Fatal(err)
	} else if string(v) != "hello world2" {
		t.Fatal(string(v))
	}

	db.Delete(key1)
}

func checkIterator(it *Iterator, cv ...int) error {
	v := make([]string, 0, len(cv))
	for ; it.Valid(); it.Next() {
		k := it.Key()
		v = append(v, string(k))
	}

	it.Close()

	if len(v) != len(cv) {
		return fmt.Errorf("len error %d != %d", len(v), len(cv))
	}

	for k, i := range cv {
		if fmt.Sprintf("key_%d", i) != v[k] {
			return fmt.Errorf("%s, %d", v[k], i)
		}
	}

	return nil
}

func TestIterator(t *testing.T) {
	db := getTestDB()

	db.Clear()

	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key_%d", i))
		value := []byte("")
		db.Put(key, value)
	}

	var it *Iterator

	k := func(i int) []byte {
		return []byte(fmt.Sprintf("key_%d", i))
	}

	it = db.Iterator(k(1), k(5), RangeClose, 0, -1)
	if err := checkIterator(it, 1, 2, 3, 4, 5); err != nil {
		t.Fatal(err)
	}

	it = db.Iterator(k(1), k(5), RangeClose, 1, 3)
	if err := checkIterator(it, 2, 3, 4); err != nil {
		t.Fatal(err)
	}

	it = db.Iterator(k(1), k(5), RangeLOpen, 0, -1)
	if err := checkIterator(it, 2, 3, 4, 5); err != nil {
		t.Fatal(err)
	}

	it = db.Iterator(k(1), k(5), RangeROpen, 0, -1)
	if err := checkIterator(it, 1, 2, 3, 4); err != nil {
		t.Fatal(err)
	}

	it = db.Iterator(k(1), k(5), RangeOpen, 0, -1)
	if err := checkIterator(it, 2, 3, 4); err != nil {
		t.Fatal(err)
	}

	it = db.RevIterator(k(1), k(5), RangeClose, 0, -1)
	if err := checkIterator(it, 5, 4, 3, 2, 1); err != nil {
		t.Fatal(err)
	}

	it = db.RevIterator(k(1), k(5), RangeClose, 1, 3)
	if err := checkIterator(it, 4, 3, 2); err != nil {
		t.Fatal(err)
	}

	it = db.RevIterator(k(1), k(5), RangeLOpen, 0, -1)
	if err := checkIterator(it, 5, 4, 3, 2); err != nil {
		t.Fatal(err)
	}

	it = db.RevIterator(k(1), k(5), RangeROpen, 0, -1)
	if err := checkIterator(it, 4, 3, 2, 1); err != nil {
		t.Fatal(err)
	}

	it = db.RevIterator(k(1), k(5), RangeOpen, 0, -1)
	if err := checkIterator(it, 4, 3, 2); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshot(t *testing.T) {
	db := getTestDB()

	key := []byte("key")
	value := []byte("hello world")

	db.Put(key, value)

	s := db.NewSnapshot()
	defer s.Close()

	db.Put(key, []byte("hello world2"))

	if v, err := s.Get(key); err != nil {
		t.Fatal(err)
	} else if string(v) != string(value) {
		t.Fatal(string(v))
	}
}

func TestDestroy(t *testing.T) {
	db := getTestDB()

	db.Destroy()

	if _, err := os.Stat(db.cfg.Path); !os.IsNotExist(err) {
		t.Fatal("must not exist")
	}
}
