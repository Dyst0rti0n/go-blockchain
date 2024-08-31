package main

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
)

// Transaction represents a transaction within the blockchain.
type Transaction struct {
	ID        string      // Unique identifier for the transaction.
	Sender    string      // Address of the sender.
	Recipient string      // Address of the recipient.
	Amount    int         // Amount of value being transferred.
	Fee       int         // Transaction fee.
	Nonce     int64       // Nonce to ensure transaction uniqueness.
	Signature *Signature  // Digital signature for the transaction.
	Timestamp int64       // Timestamp when the transaction was created.
}

// Hash generates a unique hash for the transaction based on its fields.
func (tx *Transaction) Hash() string {
	record := tx.Sender + tx.Recipient + fmt.Sprintf("%d", tx.Amount) + fmt.Sprintf("%d", tx.Fee) + fmt.Sprintf("%d", tx.Nonce)
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

// Sign signs the transaction using the sender's private key.
func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey) error {
	hash := sha256.Sum256([]byte(tx.Hash()))
	r, s, err := ecdsa.Sign(nil, privKey, hash[:])
	if err != nil {
		return err
	}
	tx.Signature = &Signature{R: r, S: s}
	return nil
}

// Verify checks if the transaction's signature is valid using the sender's public key.
func (tx *Transaction) Verify(pubKey *ecdsa.PublicKey) bool {
	if tx.Signature == nil {
		return false
	}
	hash := sha256.Sum256([]byte(tx.Hash()))
	return ecdsa.Verify(pubKey, hash[:], tx.Signature.R, tx.Signature.S)
}

// Validate ensures the transaction is valid by checking the sender's account and UTXOs.
func (tx *Transaction) Validate(accounts map[string]*Account, utxoSet *UTXOSet) error {
	if err := tx.validateAccounts(accounts); err != nil {
		return err
	}
	if err := tx.ValidateUTXO(utxoSet); err != nil {
		return err
	}
	return nil
}

// validateAccounts checks if the sender's account exists and has sufficient balance.
func (tx *Transaction) validateAccounts(accounts map[string]*Account) error {
	senderAccount, exists := accounts[tx.Sender]
	if !exists {
		return errors.New("sender account does not exist")
	}

	if senderAccount.Balance < tx.Amount+tx.Fee {
		return errors.New("insufficient balance")
	}

	return nil
}

// ValidateUTXO verifies the transaction's UTXOs and updates the UTXO set.
func (tx *Transaction) ValidateUTXO(utxoSet *UTXOSet) error {
	utxos, total := utxoSet.FindUTXOs(tx.Sender, tx.Amount+tx.Fee)
	if total < tx.Amount+tx.Fee {
		return errors.New("insufficient UTXOs")
	}

	utxoSet.SpendUTXOs(utxos)

	// Add a new UTXO for the recipient.
	newUTXO := UTXO{
		TxID:   tx.Hash(),
		Index:  0,
		Amount: tx.Amount,
		Owner:  tx.Recipient,
	}
	utxoSet.AddUTXO(newUTXO)

	// If there's change, create a UTXO for the sender.
	if change := total - (tx.Amount + tx.Fee); change > 0 {
		changeUTXO := UTXO{
			TxID:   tx.Hash(),
			Index:  1,
			Amount: change,
			Owner:  tx.Sender,
		}
		utxoSet.AddUTXO(changeUTXO)
	}

	return nil
}

// DistributeFees assigns the transaction fees to the miner.
func (tx *Transaction) DistributeFees(utxoSet *UTXOSet, minerAddress string) {
	feeUTXO := UTXO{
		TxID:   tx.Hash(),
		Index:  2,
		Amount: tx.Fee,
		Owner:  minerAddress,
	}
	utxoSet.AddUTXO(feeUTXO)
}

// Size calculates the size of the transaction in bytes.
func (tx *Transaction) Size() int {
	data, err := tx.Serialize() // Use Serialize for calculating size.
	if err != nil {
		return 0 // Handle the error appropriately if serialization fails.
	}
	return len(data)
}

// TransactionPool manages a pool of unconfirmed transactions.
type TransactionPool struct {
	transactions []*Transaction // List of transactions in the pool.
	lock         sync.Mutex     // Mutex to ensure thread-safe access.
}

// AddTransaction validates and adds a new transaction to the pool.
func (tp *TransactionPool) AddTransaction(tx *Transaction, accounts map[string]*Account, utxoSet *UTXOSet) error {
	tp.lock.Lock()
	defer tp.lock.Unlock()
	if err := tx.Validate(accounts, utxoSet); err != nil {
		return err
	}
	tp.transactions = append(tp.transactions, tx)
	tp.sortTransactionsByFee() // Sort transactions by fee for prioritization.
	return nil
}

// RemoveTransaction removes a transaction from the pool.
func (tp *TransactionPool) RemoveTransaction(tx *Transaction) {
	tp.lock.Lock()
	defer tp.lock.Unlock()
	for i, memTx := range tp.transactions {
		if memTx.Hash() == tx.Hash() {
			tp.transactions = append(tp.transactions[:i], tp.transactions[i+1:]...)
			break
		}
	}
}

// GetTransactions retrieves all transactions from the pool.
func (tp *TransactionPool) GetTransactions() []*Transaction {
	tp.lock.Lock()
	defer tp.lock.Unlock()
	return tp.transactions
}

// sortTransactionsByFee sorts the transactions by their fee in descending order.
func (tp *TransactionPool) sortTransactionsByFee() {
	sort.SliceStable(tp.transactions, func(i, j int) bool {
		return tp.transactions[i].Fee > tp.transactions[j].Fee
	})
}
