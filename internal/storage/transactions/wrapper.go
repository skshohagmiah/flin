package transactions

import (
	"github.com/dgraph-io/badger/v4"
)

// ReadTxn wraps a view transaction with common error handling
// Usage: transactions.ReadTxn(db, func(txn) { ... })
func ReadTxn(db *badger.DB, fn func(*badger.Txn) error) error {
	return db.View(fn)
}

// WriteTxn wraps an update transaction with common error handling
// Usage: transactions.WriteTxn(db, func(txn) { ... })
func WriteTxn(db *badger.DB, fn func(*badger.Txn) error) error {
	return db.Update(fn)
}

// GetKey is a helper to get a key's value from a transaction
func GetKey(txn *badger.Txn, key []byte) ([]byte, error) {
	item, err := txn.Get(key)
	if err != nil {
		return nil, err
	}
	return item.ValueCopy(nil)
}

// GetKeyWithDefault is a helper to get a key with a default value if not found
func GetKeyWithDefault(txn *badger.Txn, key []byte, defaultValue []byte) ([]byte, error) {
	item, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return defaultValue, nil
	}
	if err != nil {
		return nil, err
	}
	return item.ValueCopy(nil)
}

// ExistsKey is a helper to check if a key exists
func ExistsKey(txn *badger.Txn, key []byte) (bool, error) {
	_, err := txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// SetKey is a helper to set a key's value in a write transaction
func SetKey(txn *badger.Txn, key, value []byte) error {
	return txn.Set(key, value)
}

// DeleteKey is a helper to delete a key in a write transaction
func DeleteKey(txn *badger.Txn, key []byte) error {
	return txn.Delete(key)
}
