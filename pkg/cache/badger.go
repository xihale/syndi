package cache

import (
	"encoding/json"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type BadgerCache struct {
	db *badger.DB
}

type badgerValue struct {
	Value json.RawMessage `json:"value"`
	Exp   int64           `json:"exp"`
}

func NewBadgerCache(path string) (*BadgerCache, error) {
	opts := badger.DefaultOptions(path).WithLoggingLevel(badger.ERROR)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &BadgerCache{db: db}, nil
}

func (c *BadgerCache) Get(key string) (interface{}, bool) {
	var result interface{}
	err := c.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			var bv badgerValue
			if err := json.Unmarshal(val, &bv); err != nil {
				return err
			}
			if time.Now().Unix() >= bv.Exp {
				return badger.ErrKeyNotFound
			}
			return json.Unmarshal(bv.Value, &result)
		})
	})
	if err != nil {
		return nil, false
	}
	return result, true
}

func (c *BadgerCache) Set(key string, value interface{}, ttl time.Duration) {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return
	}
	bv := badgerValue{
		Value: valueJSON,
		Exp:   time.Now().Add(ttl).Unix(),
	}
	data, _ := json.Marshal(bv)
	c.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), data)
	})
}

func (c *BadgerCache) Exists(key string) bool {
	_, ok := c.Get(key)
	return ok
}

func (c *BadgerCache) Delete(key string) {
	c.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (c *BadgerCache) Clear() {
	c.db.DropAll()
}

func (c *BadgerCache) Close() error {
	return c.db.Close()
}
