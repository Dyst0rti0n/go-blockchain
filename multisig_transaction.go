// multisig_transaction.go
package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"strconv"
	"time"
)

type MultisigTransaction struct {
	Sender       string
	Recipient    string
	Amount       int
	Fee          int
	Signatures   []Signature
	RequiredSigs int
	Timestamp    int64
	ExpiresAt    int64
}

type Signature struct {
	R, S   *big.Int
	PubKey *ecdsa.PublicKey
}

// NewMultisigTransaction creates a new multisig transaction with a specified expiration time.
func NewMultisigTransaction(sender, recipient string, amount, fee, requiredSigs int, expirationDuration time.Duration) *MultisigTransaction {
	return &MultisigTransaction{
		Sender:       sender,
		Recipient:    recipient,
		Amount:       amount,
		Fee:          fee,
		RequiredSigs: requiredSigs,
		Timestamp:    time.Now().Unix(),
		ExpiresAt:    time.Now().Add(expirationDuration).Unix(),
	}
}

// Hash computes the hash of the transaction.
func (tx *MultisigTransaction) Hash() string {
	txData := tx.Sender + tx.Recipient + strconv.Itoa(tx.Amount) + strconv.Itoa(tx.Fee) + strconv.Itoa(tx.RequiredSigs) + strconv.FormatInt(tx.Timestamp, 10)
	hash := sha256.Sum256([]byte(txData))
	return hex.EncodeToString(hash[:])
}

// AddSignature adds a signature to the transaction.
func (tx *MultisigTransaction) AddSignature(privKey *ecdsa.PrivateKey) error {
	if time.Now().Unix() > tx.ExpiresAt {
		return errors.New("transaction has expired")
	}

	txHash := tx.Hash()
	r, s, err := ecdsa.Sign(rand.Reader, privKey, []byte(txHash))
	if err != nil {
		return err
	}
	tx.Signatures = append(tx.Signatures, Signature{R: r, S: s, PubKey: &privKey.PublicKey})
	return nil
}

// Verify checks if the transaction has the required number of valid signatures.
func (tx *MultisigTransaction) Verify() bool {
	if time.Now().Unix() > tx.ExpiresAt {
		return false
	}

	txHash := tx.Hash()
	validSigs := 0
	for _, sig := range tx.Signatures {
		if ecdsa.Verify(sig.PubKey, []byte(txHash), sig.R, sig.S) {
			validSigs++
			if validSigs >= tx.RequiredSigs {
				return true
			}
		}
	}
	return false
}

// ValidateUTXO validates the UTXOs used by the transaction and updates the UTXO set.
func (tx *MultisigTransaction) ValidateUTXO(utxoSet *UTXOSet) error {
	utxos, total := utxoSet.FindUTXOs(tx.Sender, tx.Amount+tx.Fee)
	if total < tx.Amount+tx.Fee {
		return errors.New("insufficient UTXOs")
	}

	// Spend the UTXOs
	utxoSet.SpendUTXOs(utxos)

	// Create new UTXO for the recipient
	newUTXO := UTXO{
		TxID:   tx.Hash(),
		Index:  0,
		Amount: tx.Amount,
		Owner:  tx.Recipient,
	}
	utxoSet.AddUTXO(newUTXO)

	// Create UTXO for the change back to the sender, if any
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