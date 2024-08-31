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

type Transaction struct {
	ID        string
	Sender    string
	Recipient string
	Amount    int
	Fee       int
	Nonce     int64
	Signature *Signature
	Timestamp int64
}

func (tx *Transaction) Hash() string {
	record := tx.Sender + tx.Recipient + fmt.Sprintf("%d", tx.Amount) + fmt.Sprintf("%d", tx.Fee) + fmt.Sprintf("%d", tx.Nonce)
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey) error {
	hash := sha256.Sum256([]byte(tx.Hash()))
	r, s, err := ecdsa.Sign(nil, privKey, hash[:])
	if err != nil {
		return err
	}
	tx.Signature = &Signature{R: r, S: s}
	return nil
}

func (tx *Transaction) Verify(pubKey *ecdsa.PublicKey) bool {
	if tx.Signature == nil {
		return false
	}
	hash := sha256.Sum256([]byte(tx.Hash()))
	return ecdsa.Verify(pubKey, hash[:], tx.Signature.R, tx.Signature.S)
}

func (tx *Transaction) Validate(accounts map[string]*Account, utxoSet *UTXOSet) error {
	if err := tx.validateAccounts(accounts); err != nil {
		return err
	}
	if err := tx.ValidateUTXO(utxoSet); err != nil {
		return err
	}
	return nil
}

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

func (tx *Transaction) ValidateUTXO(utxoSet *UTXOSet) error {
	utxos, total := utxoSet.FindUTXOs(tx.Sender, tx.Amount+tx.Fee)
	if total < tx.Amount+tx.Fee {
		return errors.New("insufficient UTXOs")
	}

	utxoSet.SpendUTXOs(utxos)

	newUTXO := UTXO{
		TxID:   tx.Hash(),
		Index:  0,
		Amount: tx.Amount,
		Owner:  tx.Recipient,
	}
	utxoSet.AddUTXO(newUTXO)

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

func (tx *Transaction) DistributeFees(utxoSet *UTXOSet, minerAddress string) {
	feeUTXO := UTXO{
		TxID:   tx.Hash(),
		Index:  2,
		Amount: tx.Fee,
		Owner:  minerAddress,
	}
	utxoSet.AddUTXO(feeUTXO)
}

func (tx *Transaction) Size() int {
	data, err := tx.Serialize() // Use Serialize for calculating size
	if err != nil {
		return 0 // or handle the error accordingly
	}
	return len(data)
}

type TransactionPool struct {
	transactions []*Transaction
	lock         sync.Mutex
}

func (tp *TransactionPool) AddTransaction(tx *Transaction, accounts map[string]*Account, utxoSet *UTXOSet) error {
	tp.lock.Lock()
	defer tp.lock.Unlock()
	if err := tx.Validate(accounts, utxoSet); err != nil {
		return err
	}
	tp.transactions = append(tp.transactions, tx)
	tp.sortTransactionsByFee()
	return nil
}

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

func (tp *TransactionPool) GetTransactions() []*Transaction {
	tp.lock.Lock()
	defer tp.lock.Unlock()
	return tp.transactions
}

func (tp *TransactionPool) sortTransactionsByFee() {
	sort.SliceStable(tp.transactions, func(i, j int) bool {
		return tp.transactions[i].Fee > tp.transactions[j].Fee
	})
}
