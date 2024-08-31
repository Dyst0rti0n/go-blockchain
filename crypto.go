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
	// Generate a hash of the transaction data to sign
	txHash := sha256.Sum256([]byte(tx.Sender + tx.Recipient + strconv.Itoa(tx.Amount) + strconv.Itoa(tx.Fee)))
	r, s, err = ecdsa.Sign(rand.Reader, privKey, txHash[:])
	if err != nil {
		return nil, nil, fmt.Errorf("error signing transaction: %v", err)
	}
	return r, s, nil
}

// VerifyTransaction verifies the signature of a transaction.
// This function checks if the transaction was signed by the owner of the corresponding public key.
func VerifyTransaction(tx *Transaction, r, s *big.Int, pubKey *ecdsa.PublicKey) bool {
	// Recreate the hash of the transaction data to verify the signature
	txHash := sha256.Sum256([]byte(tx.Sender + tx.Recipient + strconv.Itoa(tx.Amount) + strconv.Itoa(tx.Fee)))
	return ecdsa.Verify(pubKey, txHash[:], r, s)
}

// EncryptData encrypts data for secure communication using a symmetric key algorithm (e.g., AES).
// AES encryption is commonly used for securely transmitting data.
func EncryptData(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("error creating AES cipher: %v", err)
	}
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize] // Initialization Vector (IV) for the cipher
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("error generating IV: %v", err)
	}
	stream := cipher.NewCFBEncrypter(block, iv) // CFB mode encrypts the plaintext.
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
	return ciphertext, nil
}

// DecryptData decrypts data that was encrypted using a symmetric key algorithm (e.g., AES).
// It reverses the encryption process, returning the original plaintext.
func DecryptData(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("error creating AES cipher: %v", err)
	}
	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize] // Extract the IV from the beginning of the ciphertext.
	ciphertext = ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv) // CFB mode decrypts the ciphertext.
	stream.XORKeyStream(ciphertext, ciphertext)
	return ciphertext, nil
}
