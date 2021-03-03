package cache

import (
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/vmihailenco/msgpack/v5"
)

var (
	emptyKey = []byte("emptyCacheKey")
)

type (
	// Store _
	Store struct {
		db *badger.DB
	}
)

// NewStore _
func NewStore() (*Store, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	opt := badger.DefaultOptions(filepath.Join(home, ".cache", "honey-cachedb"))
	opt.Logger = logrus.WithField("where", "store")
	db, err := badger.Open(opt)
	if err != nil {
		return nil, err
	}

	return &Store{
		db: db,
	}, nil
}

// Close _
func (s *Store) Close() error {
	return s.db.Close()
}

// Put _
func (s *Store) Put(bucket string, key []byte, value interface{}, ttl uint32) error {
	if err := s.db.Update(func(txn *badger.Txn) error {
		data, err := msgpack.Marshal(value)
		if err != nil {
			return err
		}

		e := badger.NewEntry(append([]byte(bucket), cacheKeyName(key)...), data).WithTTL(600 * time.Second)
		return txn.SetEntry(e)
	}); err != nil {
		return err
	}

	return nil
}

// Get _
func (s *Store) Get(bucket string, key []byte, v interface{}) error {
	var value []byte

	if err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(append([]byte(bucket), key...))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			value = append([]byte{}, val...)

			return nil
		})
	}); err != nil {
		return err
	}

	return msgpack.Unmarshal(value, v)
}

func cacheKeyName(key []byte) []byte {
	if len(key) > 0 {
		return key
	}

	return emptyKey
}
