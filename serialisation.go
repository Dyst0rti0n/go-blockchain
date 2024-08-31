// serialisation.go
package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sync"
)

// transactionPool is a sync.Pool that efficiently reuses Transaction objects to minimize memory allocations.
var transactionPool = sync.Pool{
	New: func() interface{} {
		return new(Transaction)
	},
}

// Serialize encodes the transaction into a byte slice using the gob encoding format.
func (tx *Transaction) Serialize() ([]byte, error) {
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}
	return encoded.Bytes(), nil
}

// DeserializeTransaction decodes a byte slice into a Transaction object, reusing objects from the pool.
func DeserializeTransaction(data []byte) (*Transaction, error) {
	tx := transactionPool.Get().(*Transaction)
	defer transactionPool.Put(tx) // Return the transaction object to the pool after deserialization.

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %w", err)
	}
	return tx, nil
}
