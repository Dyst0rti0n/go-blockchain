// proof_of_work.go
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ProofOfWork struct {
	Block      *Block
	Difficulty int
}

func NewProofOfWork(b *Block) *ProofOfWork {
	return &ProofOfWork{
		Block:      b,
		Difficulty: b.Difficulty,
	}
}

// Run performs the proof of work using concurrency and includes a timeout mechanism.
func (pow *ProofOfWork) Run() (int, string, error) {
    var wg sync.WaitGroup
    var mu sync.Mutex
    found := false
    var nonce int
    var hash string

    numWorkers := runtime.NumCPU() // Number of goroutines for parallel computation based on CPU cores
    workChan := make(chan int, numWorkers)

    timeout := time.After(5 * time.Minute) // Set a timeout for the mining process

    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for n := range workChan {
                h := pow.calculateHash(n)
                if strings.HasPrefix(h, strings.Repeat("0", pow.Difficulty)) {
                    mu.Lock()
                    if !found {
                        found = true
                        nonce = n
                        hash = h
                        close(workChan)
                    }
                    mu.Unlock()
                    break
                }
            }
        }()
    }

    rand.Seed(time.Now().UnixNano())
    startNonce := rand.Intn(1_000_000_000)
    go func() {
        for i := startNonce; !found; i++ {
            select {
            case <-timeout:
                close(workChan)
                return
            default:
                workChan <- i
            }
        }
    }()
    wg.Wait()

    if !found {
        return 0, "", errors.New("proof of work failed: timeout reached")
    }

    return nonce, hash, nil
}

func (pow *ProofOfWork) calculateHash(nonce int) string {
	record := strconv.Itoa(pow.Block.Index) +
		strconv.FormatInt(pow.Block.Timestamp, 10) +
		pow.Block.PreviousHash +
		pow.Block.calculateMerkleRoot() +
		strconv.Itoa(nonce) +
		strconv.Itoa(pow.Difficulty)
	hash := sha256.Sum256([]byte(record))
	return hex.EncodeToString(hash[:])
}

// Validate checks if the provided nonce results in a valid hash.
func (pow *ProofOfWork) Validate() bool {
	hash := pow.calculateHash(pow.Block.Nonce)
	return strings.HasPrefix(hash, strings.Repeat("0", pow.Difficulty))
}

