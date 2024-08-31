// token.go
package main

import (
	"errors"
	"log"
	"sync"
)

type Token struct {
	balances sync.Map // Use sync.Map for better concurrency performance
	lock     sync.RWMutex
}

func NewToken() *Token {
	return &Token{}
}

func (t *Token) BalanceOf(address string) int {
	value, _ := t.balances.Load(address)
	if balance, ok := value.(int); ok {
		return balance
	}
	return 0
}

// Transfer handles the movement of tokens between accounts with event hooks.
func (t *Token) Transfer(from, to string, amount int) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	fromBalance, _ := t.balances.LoadOrStore(from, 0)
	toBalance, _ := t.balances.LoadOrStore(to, 0)

	if fromBalance.(int) < amount {
		return errors.New("insufficient balance")
	}

	// Trigger a pre-transfer hook (e.g., for logging or analytics)
	t.preTransferHook(from, to, amount)

	t.balances.Store(from, fromBalance.(int)-amount)
	t.balances.Store(to, toBalance.(int)+amount)

	// Trigger a post-transfer hook (e.g., for triggering smart contracts)
	t.postTransferHook(from, to, amount)

	return nil
}

// Mint creates new tokens and adds them to the specified address.
func (t *Token) Mint(address string, amount int) {
	t.lock.Lock()
	defer t.lock.Unlock()

	currentBalance, _ := t.balances.LoadOrStore(address, 0)
	t.balances.Store(address, currentBalance.(int)+amount)
}

// Burn allows for tokens to be permanently destroyed.
func (t *Token) Burn(address string, amount int) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	currentBalance, _ := t.balances.LoadOrStore(address, 0)

	if currentBalance.(int) < amount {
		return errors.New("insufficient balance to burn")
	}

	t.balances.Store(address, currentBalance.(int)-amount)
	return nil
}

// Hooks for custom behavior before and after a transfer.
func (t *Token) preTransferHook(from, to string, amount int) {
	log.Printf("Pre-Transfer Hook: %s is transferring %d tokens to %s\n", from, amount, to)
}

func (t *Token) postTransferHook(from, to string, amount int) {
	log.Printf("Post-Transfer Hook: %s has transferred %d tokens to %s\n", from, amount, to)
}
