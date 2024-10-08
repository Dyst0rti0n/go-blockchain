// utxo.go
package main

import (
	"sync"
)

// UTXO represents an unspent transaction output in the blockchain.
type UTXO struct {
	TxID   string // Transaction ID where this UTXO originates.
	Index  int    // Index of the UTXO in the transaction.
	Amount int    // Amount of value this UTXO represents.
	Owner  string // Address of the UTXO owner.
}

// UTXOSet maintains a set of all unspent transaction outputs.
type UTXOSet struct {
	UTXOs map[string]map[int]UTXO // Nested map for quick lookup by TxID and Index.
	lock  sync.RWMutex            // RWMutex for thread-safe access to the UTXO set.
}

func NewUTXOSet() *UTXOSet {
	return &UTXOSet{
		UTXOs: make(map[string]map[int]UTXO),
	}
}

// FindUTXOs finds unspent transaction outputs (UTXOs) for a given owner and amount.
func (u *UTXOSet) FindUTXOs(owner string, amount int) ([]UTXO, int) {
	u.lock.RLock()
	defer u.lock.RUnlock()

	var accumulated []UTXO
	accumulatedValue := 0

	for _, outputs := range u.UTXOs {
		for _, utxo := range outputs {
			if utxo.Owner == owner {
				accumulated = append(accumulated, utxo)
				accumulatedValue += utxo.Amount
				if accumulatedValue >= amount {
					return accumulated, accumulatedValue
				}
			}
		}
	}

	return accumulated, accumulatedValue
}

// SpendUTXOs marks the given UTXOs as spent by removing them from the set.
func (u *UTXOSet) SpendUTXOs(utxos []UTXO) {
	u.lock.Lock()
	defer u.lock.Unlock()

	for _, spent := range utxos {
		if outputs, exists := u.UTXOs[spent.TxID]; exists {
			delete(outputs, spent.Index)
			if len(outputs) == 0 {
				delete(u.UTXOs, spent.TxID)
			}
		}
	}
}

// AddUTXO adds a new unspent transaction output to the UTXO set.
func (u *UTXOSet) AddUTXO(utxo UTXO) {
	u.lock.Lock()
	defer u.lock.Unlock()

	if _, exists := u.UTXOs[utxo.TxID]; !exists {
		u.UTXOs[utxo.TxID] = make(map[int]UTXO)
	}
	u.UTXOs[utxo.TxID][utxo.Index] = utxo
}

// HasUTXO checks if the given owner has any UTXOs in the set.
func (u *UTXOSet) HasUTXO(owner string) bool {
	u.lock.RLock()
	defer u.lock.RUnlock()

	for _, outputs := range u.UTXOs {
		for _, utxo := range outputs {
			if utxo.Owner == owner {
				return true
			}
		}
	}
	return false
}

// GetBalance returns the total balance for a given owner by summing all their UTXOs.
func (u *UTXOSet) GetBalance(owner string) int {
	u.lock.RLock()
	defer u.lock.RUnlock()

	balance := 0
	for _, outputs := range u.UTXOs {
		for _, utxo := range outputs {
			if utxo.Owner == owner {
				balance += utxo.Amount
			}
		}
	}
	return balance
}
