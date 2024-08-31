package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// Account struct represents a basic amount with an address, balance, nonce and public key. 
type Account struct {
	Address   string
	Balance   int
	Nonce     int64
	PublicKey *ecdsa.PublicKey
}

// NewAccount creates a new account with the provided address, initial balance, and public key.
// This is like setting up a new bank account with a starting balance.
func NewAccount(address string, initialBalance int, publicKey *ecdsa.PublicKey) *Account {
	return &Account{
		Address:   address,
		Balance:   initialBalance,
		Nonce:     0,				// Nonce starts at zero and increments with each transaction
		PublicKey: publicKey,
	}
}

// This increases the nonce by one. This is crucial to prevent replay attacks in transactions. 
func (acc *Account) IncrementNonce() {
	acc.Nonce++
}

// Debit reduces the account balance by the specified amount, but only if there's enough abalance. 
func (acc *Account) Debit(amount int) error {
	if acc.Balance < amount {
		return fmt.Errorf("insufficient balance")
	}
	acc.Balance -= amount
	return nil
}

// Adds the specified amount to the acc bal
func (acc *Account) Credit(amount int) {
	acc.Balance += amount
}

// Represents a user's wallet containing keys and an address.
type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
	Address    string
}

// Generates a new wallet. It creates a new key pair and derives an address from the public key.
func NewWallet() (*Wallet, error) {
	privKey, pubKey, err := GenerateKeyPair()
	if err != nil {
		return nil, err
	}

	// Convert the public key to a format that can be easily stored and retrieved (PEM format) 
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	// The address is just a hex-encoded version of the public key bytes.
	address := hex.EncodeToString(pubKeyBytes)

	return &Wallet{
		PrivateKey: privKey,
		PublicKey:  pubKey,
		Address:    address,
	}, nil
}

// SaveToFile saves the wallet to a file using the new os and encoding packages.
func (w *Wallet) SaveToFile(filename string) error {
	privBytes, err := x509.MarshalECPrivateKey(w.PrivateKey)
	if err != nil {
		return err
	}
	privBlock := &pem.Block{
		Type:  "EC PRIVATE KEY", 		// Specifies that this is an EC private key
		Bytes: privBytes,
	}
	// Save the PEM-encoded private key to a file, making sure the permissions are secure.
	return os.WriteFile(filename, pem.EncodeToMemory(privBlock), 0600)
}

// LoadWallet loads a wallet from a file. It reads the provide key and reconstructs the wallet.
func LoadWallet(filename string) (*Wallet, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Decodes the PEM block containing the private key
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "EC PRIVATE KEY" {
		return nil, errors.New("failed to decode PEM block containing the private key")
	}

	// Parse the private key from the block
	privKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	pubKey := &privKey.PublicKey

	// Convert the public key to a format that can be easily stored and retrieved. 
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, err
	}

	// Recreate the wallet's address from the public key
	address := hex.EncodeToString(pubKeyBytes)

	return &Wallet{
		PrivateKey: privKey,
		PublicKey:  pubKey,
		Address:    address,
	}, nil
}
