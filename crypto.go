package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"
)

// GenerateKeyPair generates a new ECDSA key pair using the P-256 elliptic curve.
func GenerateKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("error generating ECDSA key pair: %v", err)
	}
	return privateKey, &privateKey.PublicKey, nil
}

// SignTransaction signs a transaction using the given private key.
func SignTransaction(tx *Transaction, privKey *ecdsa.PrivateKey) (r, s *big.Int, err error) {
	txHash := sha256.Sum256([]byte(tx.Sender + tx.Recipient + strconv.Itoa(tx.Amount) + strconv.Itoa(tx.Fee)))
	r, s, err = ecdsa.Sign(rand.Reader, privKey, txHash[:])
	if err != nil {
		return nil, nil, fmt.Errorf("error signing transaction: %v", err)
	}
	return r, s, nil
}

// VerifyTransaction verifies the signature of a transaction.
func VerifyTransaction(tx *Transaction, r, s *big.Int, pubKey *ecdsa.PublicKey) bool {
	txHash := sha256.Sum256([]byte(tx.Sender + tx.Recipient + strconv.Itoa(tx.Amount) + strconv.Itoa(tx.Fee)))
	return ecdsa.Verify(pubKey, txHash[:], r, s)
}

// EncryptData encrypts data for secure communication using a symmetric key algorithm (e.g., AES).
func EncryptData(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("error creating AES cipher: %v", err)
	}
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("error generating IV: %v", err)
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
	return ciphertext, nil
}

// DecryptData decrypts data for secure communication using a symmetric key algorithm (e.g., AES).
func DecryptData(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("error creating AES cipher: %v", err)
	}
	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return ciphertext, nil
}
