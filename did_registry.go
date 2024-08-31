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

// DID represents a Decentralized Identifier, a unique ID with associated public key and attributes.
type DID struct {
	ID         string
	PublicKey  string
	Owner      string
	Attributes map[string]string // Additional attributes or claims associated with the DID
	CreatedAt  int64
}

// DIDRegistry is a registry that manages DIDs, allowing for their registration, resolution, and authentication.
type DIDRegistry struct {
	dids map[string]*DID
	lock sync.RWMutex // Ensures thread-safe operations on the DID registry.
}

// NewDIDRegistry initializes a new, empty DID registry.
func NewDIDRegistry() *DIDRegistry {
	return &DIDRegistry{
		dids: make(map[string]*DID),
	}
}

// RegisterDID registers a new DID with an owner, public key, and optional attributes.
// This creates a new identifier that can be used for decentralized authentication.
func (dr *DIDRegistry) RegisterDID(owner, publicKey string, attributes map[string]string) (string, error) {
	dr.lock.Lock()
	defer dr.lock.Unlock()

	didID := generateDIDID() // Generate a unique ID for the DID.
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

// ResolveDID retrieves a DID from the registry based on its ID.
// This function allows others to lookup the public key and attributes associated with a DID.
func (dr *DIDRegistry) ResolveDID(didID string) (*DID, error) {
	dr.lock.RLock()
	defer dr.lock.RUnlock()

	did, exists := dr.dids[didID]
	if !exists {
		return nil, fmt.Errorf("DID not found")
	}
	return did, nil
}

// AuthenticateDID verifies a DID's signature against a given message, proving the owner's identity.
func (dr *DIDRegistry) AuthenticateDID(didID, signature, message string) (bool, error) {
	dr.lock.RLock()
	defer dr.lock.RUnlock()

	did, exists := dr.dids[didID]
	if !exists {
		return false, fmt.Errorf("DID not found")
	}

	// Verify the signature using the DID's public key.
	isValid := verifySignature(did.PublicKey, signature, message)
	return isValid, nil
}

// verifySignature checks if a signature is valid by using the provided public key and message.
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

// generateDIDID generates a unique identifier for a DID using the current timestamp.
func generateDIDID() string {
	return fmt.Sprintf("did:example:%x", sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))))
}
