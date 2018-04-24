package iostdb

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

var test_values = []string{"", "a", "123", "\x00"}

func TestMemDB_PutGet(t *testing.T) {
	db, _ := NewMemDatabase()
	testPutGet(db, t)
}

func TestLDB_PutGet(t *testing.T) {
	dirname, _ := ioutil.TempDir(os.TempDir(), "ethdb_test_")
	db, _ := NewLDBDatabase(dirname, 0, 0)
	testPutGet(db, t)
}

func testPutGet(db Database, t *testing.T) {
	t.Parallel()

	for _, v := range test_values {
		err := db.Put([]byte(v), []byte(v))
		if err != nil {
			t.Fatalf("put failed: %v", err)
		}
	}
	for _, v := range test_values {
		data, err := db.Get([]byte(v))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if !bytes.Equal(data, []byte(v)) {
			t.Fatalf("get returned wrong result, got %q", string(data))
		}
	}
	for _, v := range test_values {
		bo, err := db.Has([]byte(v))
		if err != nil {
			t.Fatalf("has failed: %v", err)
		}
		if !bo {
			t.Fatalf("has returned wrong result failed")

		}
	}
	for _, v := range test_values {
		err := db.Put([]byte(v), []byte("test"))
		if err != nil {
			t.Fatalf("put failed: %v", err)
		}
	}
}