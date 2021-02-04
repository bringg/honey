package cache

import (
	"encoding/json"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/xujiajun/nutsdb"
)

type (
	// Store _
	Store struct {
		db *nutsdb.DB
	}
)

// NewStore _
func NewStore() (*Store, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	opt := nutsdb.DefaultOptions
	opt.Dir = filepath.Join(home, ".cache", "honey-cachedb")
	db, err := nutsdb.Open(opt)
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
func (s *Store) Put(bucket string, key []byte, value interface{}) error {
	if err := s.db.Update(
		func(tx *nutsdb.Tx) error {
			data, err := json.Marshal(value)
			if err != nil {
				return err
			}

			// If set ttl = 0 or Persistent, this key will nerver expired.
			// Set ttl = 600 , after 600 seconds, this key will expired.
			if err := tx.Put(bucket, key, data, 600); err != nil {
				return err
			}

			return nil
		}); err != nil {
		return err
	}

	return nil
}

// Get _
func (s *Store) Get(bucket string, key []byte, v interface{}) error {
	var value []byte

	if err := s.db.View(
		func(tx *nutsdb.Tx) error {
			e, err := tx.Get(bucket, key)
			if err != nil {
				return err
			}

			value = e.Value

			return nil
		}); err != nil {
		return err
	}

	if err := json.Unmarshal(value, v); err != nil {
		return err
	}

	return nil
}
