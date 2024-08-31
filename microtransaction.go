package main

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Microtransaction represents a small transaction
type Microtransaction struct {
	ID         string
	Sender     string
	Recipient  string
	Amount     int64
	Fee        int64
	Timestamp  int64
	Signature  *Signature
	BatchID    string
}

// MicrotransactionBatch represents a batch of microtransactions
type MicrotransactionBatch struct {
	ID               string
	Transactions     []*Microtransaction
	TotalAmount      int64
	TotalFees        int64
	Processed        bool
	CreatedAt        int64
	ProcessingNode   string
	BatchReward      int64
	ProcessingStatus string
}

type MicrotransactionPool struct {
	Transactions map[string]*Microtransaction
	Batches      map[string]*MicrotransactionBatch
	lock         sync.RWMutex
}

func NewMicrotransactionPool() *MicrotransactionPool {
	return &MicrotransactionPool{
		Transactions: make(map[string]*Microtransaction),
		Batches:      make(map[string]*MicrotransactionBatch),
	}
}

// AddMicrotransaction adds a microtransaction to the pool
func (mp *MicrotransactionPool) AddMicrotransaction(tx *Microtransaction) error {
	mp.lock.Lock()
	defer mp.lock.Unlock()

	tx.ID = generateTransactionID()

	// Check for transaction uniqueness
	if _, exists := mp.Transactions[tx.ID]; exists {
		return fmt.Errorf("transaction with ID %s already exists", tx.ID)
	}

	mp.Transactions[tx.ID] = tx
	return nil
}

// CreateBatch creates a new batch of microtransactions for processing
func (mp *MicrotransactionPool) CreateBatch() *MicrotransactionBatch {
	mp.lock.Lock()
	defer mp.lock.Unlock()

	if len(mp.Transactions) == 0 {
		return nil
	}

	batchID := generateBatchID()
	batch := &MicrotransactionBatch{
		ID:           batchID,
		CreatedAt:    time.Now().Unix(),
		Transactions: make([]*Microtransaction, 0, len(mp.Transactions)),
	}

	// Calculate total amount and fees for the batch
	for _, tx := range mp.Transactions {
		tx.BatchID = batchID
		batch.TotalAmount += tx.Amount
		batch.TotalFees += tx.Fee
		batch.Transactions = append(batch.Transactions, tx)
	}

	// Clear the current transaction pool
	mp.Transactions = make(map[string]*Microtransaction)
	mp.Batches[batchID] = batch
	return batch
}

// ProcessBatch processes a batch of microtransactions
func (mp *MicrotransactionPool) ProcessBatch(batchID, nodeAddress string) error {
	mp.lock.Lock()
	defer mp.lock.Unlock()

	batch, exists := mp.Batches[batchID]
	if !exists {
		return fmt.Errorf("batch %s not found", batchID)
	}

	if batch.Processed {
		return fmt.Errorf("batch %s already processed", batchID)
	}

	batch.Processed = true
	batch.ProcessingNode = nodeAddress
	batch.ProcessingStatus = "Success"
	batch.BatchReward = batch.TotalFees / 2 // Reward node with half the fees

	return nil
}

// DistributeTippingReward distributes the rewards from a batch to the recipient
func (mp *MicrotransactionPool) DistributeTippingReward(batch *MicrotransactionBatch, accounts map[string]*Account) {
	mp.lock.Lock()
	defer mp.lock.Unlock()

	for _, tx := range batch.Transactions {
		account, exists := accounts[tx.Recipient]
		if !exists {
			// Provide the correct number of arguments for NewAccount
			account = NewAccount(tx.Recipient, int(tx.Amount), nil) // Assuming public key is not used here
			accounts[tx.Recipient] = account
		}
		// Convert int64 to int for the Credit method
		account.Credit(int(tx.Amount))
	}
}

// SignMicrotransaction signs the microtransaction with the sender's private key
func (tx *Microtransaction) Sign(privKey *ecdsa.PrivateKey) error {
	hash := tx.Hash()
	r, s, err := ecdsa.Sign(nil, privKey, []byte(hash))
	if err != nil {
		return err
	}
	tx.Signature = &Signature{R: r, S: s}
	return nil
}

// VerifyMicrotransaction verifies the transaction's signature
func (tx *Microtransaction) Verify(pubKey *ecdsa.PublicKey) bool {
	if tx.Signature == nil {
		return false
	}
	hash := tx.Hash()
	return ecdsa.Verify(pubKey, []byte(hash), tx.Signature.R, tx.Signature.S)
}

// Hash generates a hash of the transaction
func (tx *Microtransaction) Hash() string {
	record := tx.Sender + tx.Recipient + fmt.Sprintf("%d", tx.Amount) + fmt.Sprintf("%d", tx.Fee) + fmt.Sprintf("%d", tx.Timestamp)
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

// generateTransactionID generates a unique ID for a transaction
func generateTransactionID() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))))
}

// generateBatchID generates a unique ID for a batch
func generateBatchID() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("batch-%d", time.Now().UnixNano()))))
}
