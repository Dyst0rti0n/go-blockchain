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

// ProofOfWork represents the proof of work algorithm used to secure the blockchain.
type ProofOfWork struct {
	Block      *Block  // The block that is being mined.
	Difficulty int     // The difficulty level for mining, represented by the number of leading zeros required in the hash.
}

func NewProofOfWork(b *Block) *ProofOfWork {
	return &ProofOfWork{
		Block:      b,
		Difficulty: b.Difficulty,
	}
}

// Run performs the proof of work using concurrency and includes a timeout mechanism.
// It tries to find a nonce that results in a hash with the required number of leading zeros.
func (pow *ProofOfWork) Run() (int, string, error) {
    var wg sync.WaitGroup
    var mu sync.Mutex
    found := false
    var nonce int
    var hash string

    numWorkers := runtime.NumCPU() // Determine the number of goroutines based on available CPU cores.
    workChan := make(chan int, numWorkers)

    timeout := time.After(5 * time.Minute) // Set a timeout for the mining process.

    randGen := rand.New(rand.NewSource(time.Now().UnixNano())) // Updated to use new source for better predictability.
    startNonce := randGen.Intn(1_000_000_000)

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
                        close(workChan) // Stop other goroutines once the solution is found.
                    }
                    mu.Unlock()
                    break
                }
            }
        }()
    }

    go func() {
        for i := startNonce; !found; i++ {
            select {
            case <-timeout:
                close(workChan) // Stop all work if the timeout is reached.
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

// calculateHash generates a SHA-256 hash of the block's data combined with the given nonce.
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

// Validate checks if the provided nonce results in a valid hash that meets the difficulty criteria.
func (pow *ProofOfWork) Validate() bool {
	hash := pow.calculateHash(pow.Block.Nonce)
	return strings.HasPrefix(hash, strings.Repeat("0", pow.Difficulty))
}
