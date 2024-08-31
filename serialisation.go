package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sync"
)

var transactionPool = sync.Pool{
	New: func() interface{} {
		return new(Transaction)
	},
}

func (tx *Transaction) Serialize() ([]byte, error) {
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize transaction: %w", err)
	}
	return encoded.Bytes(), nil
}

func DeserializeTransaction(data []byte) (*Transaction, error) {
	tx := transactionPool.Get().(*Transaction)
	defer transactionPool.Put(tx)

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %w", err)
	}
	return tx, nil
}
