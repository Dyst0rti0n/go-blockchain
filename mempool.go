package main

import (
	"errors"
	"sort"
	"sync"
	"time"
)

// Mempool is a pool that holds transactions before they are confirmed and added to a block.
type Mempool struct {
	transactions map[string]*Transaction // Using a map for quick lookups and uniqueness
	lock         sync.RWMutex            // Read-write lock for thread-safe access
}

// Initialises a new Mempool
func NewMempool() *Mempool {
	return &Mempool{
		transactions: make(map[string]*Transaction),
	}
}

// Adds a new transaction to the mempool after validating it.
func (m *Mempool) AddTransaction(tx *Transaction, accounts map[string]*Account, utxoSet *UTXOSet) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	// Validate the transaction before adding
	if err := tx.Validate(accounts, utxoSet); err != nil {
		return errors.New("invalid transaction: " + err.Error())
	}

	txID := tx.Hash()
	if _, exists := m.transactions[txID]; exists {
		return errors.New("transaction already exists in the mempool")
	}

	m.transactions[txID] = tx

	// Sort transactions by fee, descending order (highest fee first)
	transactions := m.GetTransactions()
	m.sortTransactionsByFee(transactions)

	return nil
}

// Removes a transaction from the mempool.
func (m *Mempool) RemoveTransaction(tx *Transaction) {
	m.lock.Lock()
	defer m.lock.Unlock()

	txID := tx.Hash()
	delete(m.transactions, txID)
}

// Returns a specific transaction by its ID
func (m *Mempool) GetTransaction(txID string) *Transaction {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.transactions[txID]
}

// Returns a list of all transactions in the mempool, sorted by fee.
func (m *Mempool) GetTransactions() []*Transaction {
	m.lock.RLock()
	defer m.lock.RUnlock()

	transactions := make([]*Transaction, 0, len(m.transactions))
	for _, tx := range m.transactions {
		transactions = append(transactions, tx)
	}

	m.sortTransactionsByFee(transactions)

	return transactions
}

// Checks if the mempool is empty.
func (m *Mempool) IsEmpty() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.transactions) == 0
}

// Clear clears the mempool, removing all transactions.
func (m *Mempool) Clear() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.transactions = make(map[string]*Transaction)
}

// Sorts the transactions by fee in descending order.
func (m *Mempool) sortTransactionsByFee(transactions []*Transaction) {
	sort.SliceStable(transactions, func(i, j int) bool {
		return transactions[i].Fee > transactions[j].Fee
	})
}

// Removes transactions that have been in the mempool for too long.
func (m *Mempool) PurgeOldTransactions(maxAge time.Duration) {
	m.lock.Lock()
	defer m.lock.Unlock()

	currentTime := time.Now().Unix()
	for txID, tx := range m.transactions {
		if currentTime-tx.Timestamp > int64(maxAge.Seconds()) {
			delete(m.transactions, txID)
		}
	}
}
