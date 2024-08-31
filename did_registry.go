package main

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
	"math/big"
)

type DID struct {
	ID         string
	PublicKey  string
	Owner      string
	Attributes map[string]string // Additional attributes or claims associated with the DID
	CreatedAt  int64
}

type DIDRegistry struct {
	dids map[string]*DID
	lock sync.RWMutex
}

func NewDIDRegistry() *DIDRegistry {
	return &DIDRegistry{
		dids: make(map[string]*DID),
	}
}

func (dr *DIDRegistry) RegisterDID(owner, publicKey string, attributes map[string]string) (string, error) {
	dr.lock.Lock()
	defer dr.lock.Unlock()

	didID := generateDIDID()
	did := &DID{
		ID:         didID,
		PublicKey:  publicKey,
		Owner:      owner,
		Attributes: attributes,
		CreatedAt:  time.Now().Unix(),
	}
	dr.dids[didID] = did

	return didID, nil
}

func (dr *DIDRegistry) ResolveDID(didID string) (*DID, error) {
	dr.lock.RLock()
	defer dr.lock.RUnlock()

	did, exists := dr.dids[didID]
	if !exists {
		return nil, fmt.Errorf("DID not found")
	}
	return did, nil
}

func (dr *DIDRegistry) AuthenticateDID(didID, signature, message string) (bool, error) {
	dr.lock.RLock()
	defer dr.lock.RUnlock()

	did, exists := dr.dids[didID]
	if !exists {
		return false, fmt.Errorf("DID not found")
	}

	isValid := verifySignature(did.PublicKey, signature, message)
	return isValid, nil
}

func verifySignature(publicKey, signature, message string) bool {
	pubKeyBytes, _ := hex.DecodeString(publicKey)
	pubKey := &ecdsa.PublicKey{}
	pubKey.X = new(big.Int).SetBytes(pubKeyBytes[:len(pubKeyBytes)/2])
	pubKey.Y = new(big.Int).SetBytes(pubKeyBytes[len(pubKeyBytes)/2:])

	sigBytes, _ := hex.DecodeString(signature)
	r := new(big.Int).SetBytes(sigBytes[:len(sigBytes)/2])
	s := new(big.Int).SetBytes(sigBytes[len(sigBytes)/2:])

	hashedMessage := sha256.Sum256([]byte(message))
	return ecdsa.Verify(pubKey, hashedMessage[:], r, s)
}

func generateDIDID() string {
	return fmt.Sprintf("did:example:%x", sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))))
}
