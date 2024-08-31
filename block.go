package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"
)

// this struct represents a single block in the bc
type Block struct {
	Index        int				// Position of the block in the chain
	Timestamp    int64				// When block was created
	PreviousHash string
	Hash         string				// Calculated hash of this block
	Transactions []*Transaction	
	Nonce        int				// Nonce used for POW
	Difficulty   int				// Mining difficulty level
}

// Constants for various bc settings
const (
	BlockReward        = 50		   // Fixed block reward for miners
	AdjustmentInterval = 10		   // How often the difficulty is adjusted
	MaxBlockSize       = 1_000_000 // Max block size in bytes for scalability
	MinTransactionFee  = 1         // Min fee for transactions
)

// Creates new block
func NewBlock(transactions []*Transaction, previousHash string, difficulty int) *Block {
	block := &Block{
		Index:        0,					// Initially set index to 0, will be set later
		Timestamp:    time.Now().Unix(),	// Record the current time as the block's timestamp
		PreviousHash: previousHash,			// Link to previous block
		Transactions: transactions,			// Add transaction
		Difficulty:   difficulty,			// Set difficulty for this block
	}
	block.Hash = block.calculateHash()		// Calculate block's hash based on its content
	return block
}

// Calculate has based on its content
func (b *Block) calculateHash() string {
	// Combine block data into a single string
	record := strconv.Itoa(b.Index) +
		strconv.FormatInt(b.Timestamp, 10) +
		b.PreviousHash +
		b.calculateMerkleRoot() +
		strconv.Itoa(b.Nonce) +
		strconv.Itoa(b.Difficulty)

	// Generate SHA-256 hash
	hash := sha256.Sum256([]byte(record))
	return hex.EncodeToString(hash[:])		// Return hash as a hex string
}

// Calculate merkle root for the block's transactions (a has of all transaction hashes)
func (b *Block) calculateMerkleRoot() string {
	var transactionHashes []string
	for _, tx := range b.Transactions {
		transactionHashes = append(transactionHashes, tx.Hash()) // Get the hash of each transaction
	}
	return calculateMerkleRoot(transactionHashes)
}

// Helper function
func calculateMerkleRoot(transactionHashes []string) string {
	if len(transactionHashes) == 0 {
		return ""
	}
	if len(transactionHashes) == 1 {
		return transactionHashes[0]
	}

	var newLevel []string
	for i := 0; i < len(transactionHashes)-1; i += 2 {
		// Combine and hash pairs of trans hashes
		hash := sha256.Sum256([]byte(transactionHashes[i] + transactionHashes[i+1]))
		newLevel = append(newLevel, hex.EncodeToString(hash[:]))
	}
	// If there's an off no. of hashes, hash the last one again 
	if len(transactionHashes)%2 == 1 {
		hash := sha256.Sum256([]byte(transactionHashes[len(transactionHashes)-1]))
		newLevel = append(newLevel, hex.EncodeToString(hash[:]))
	}
	return calculateMerkleRoot(newLevel) // Recursively caluclate until one hash remains 
}

// Blockchain struct represents the entire blockchain(bc)
type Blockchain struct {
	Blocks              []*Block			   // Array ofall blocks in the chain
	Stake               map[string]int         // Stake mapping for PoS (address to stake amount)
	blockReward         int                    // Internal value for block reward
	ProtocolVersion     string                 // Track the current protocol version
	ConsensusAlgorithm  string                 // Track the current consensus algorithm (e.g PoW, PoS)
	MaxBlockSize        int                    // Max block size allowed in bytes
	lock                sync.RWMutex           // Lock for thread-safe access
	Mempool             *Mempool               // Holds unconfirmed transactions
	Accounts            map[string]*Account    // Tracks accounts and their balances
	UTXOSet             *UTXOSet               // Manages the Unspent Transaction Outputs (UTXOs)
	ContractEngine      *ContractEngine		   // Manages smart contracts
	DIDRegistry         *DIDRegistry		   // Manages Decentralised Identifiers (DIDs)
	MinerAddress        string                 // Address of current miner
}

// Initialise a new bc, starting with the genesis block
func NewBlockchain() *Blockchain {
	genesisBlock := NewBlock([]*Transaction{}, "0", 1) 	// Genesis block with no transactions and difficulty 1
	return &Blockchain{
		Blocks:             []*Block{genesisBlock},		// Bc starts with the genesis block
		Stake:              make(map[string]int),
		blockReward:        BlockReward,				// Set initial block reward
		ProtocolVersion:    "v1.0",						// Default protocol version
		ConsensusAlgorithm: "PoW", 						// Default to Proof of Work
		MaxBlockSize:       MaxBlockSize,				// Set maximum block size
		Mempool:            NewMempool(), 				// Initialise the transaction pool
		Accounts:           make(map[string]*Account),
		UTXOSet:            NewUTXOSet(),
		ContractEngine:     NewContractEngine(),
		DIDRegistry:        NewDIDRegistry(),
	}
}

// Adjust the mining difficulty based on the time it took to mine the last blocks
func (bc *Blockchain) AdjustDifficulty() int {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	if len(bc.Blocks)%AdjustmentInterval != 0 {
		return bc.Blocks[len(bc.Blocks)-1].Difficulty		// No adjustment needed
	}

	// Calculate time tkaen to mine the last AdjustsmentInterval blocks
	lastAdjustmentBlock := bc.Blocks[len(bc.Blocks)-AdjustmentInterval]
	expectedTime := AdjustmentInterval * 10 * 60 // Assuming 10 minutes per block
	actualTime := int(bc.Blocks[len(bc.Blocks)-1].Timestamp - lastAdjustmentBlock.Timestamp)

	// Adjust difficulty based on block mining times
	if actualTime < expectedTime/2 {
		return lastAdjustmentBlock.Difficulty + 1
	} else if actualTime > expectedTime*2 {
		if lastAdjustmentBlock.Difficulty > 1 {
			return lastAdjustmentBlock.Difficulty - 1
		}
	}

	return lastAdjustmentBlock.Difficulty		// No significant change, return current difficulty
}

// Adds a new block to the bc after validating and processing the transactions
func (bc *Blockchain) AddBlock(transactions []*Transaction) *Block {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	// Adjust difficulty based on the current state of the bc
	difficulty := bc.AdjustDifficulty()
	lastBlock := bc.Blocks[len(bc.Blocks)-1]

	// If no miner address is set, select one based on stake or account balance
	if bc.MinerAddress == "" {
		bc.MinerAddress = bc.SelectMinerAddress()
	}

	// Reward the miner
	minerRewardTx := &Transaction{
		Sender:    "system",			// System generates the reward
		Recipient: bc.MinerAddress,		// Reward goes to the miner
		Amount:    bc.GetBlockReward(),	// Reward amount based on current block reward
		Fee:       0,					// No fee for reward transactions
	}
	transactions = append([]*Transaction{minerRewardTx}, transactions...)	// Reward transaction

	// Sort transactions by fee (highest fee first)
	sort.SliceStable(transactions, func(i, j int) bool {
		return transactions[i].Fee > transactions[j].Fee
	})

	// Collect valid transactions up to the max block size
	validTransactions := []*Transaction{minerRewardTx}
	currentSize := minerRewardTx.Size() 
	for _, tx := range transactions {
		if bc.IsValidTransaction(tx) {
			txSize := tx.Size()
			if currentSize+txSize <= MaxBlockSize {
				validTransactions = append(validTransactions, tx)
				currentSize += txSize
			}
		}
	}

	// Create a new block with the valid transactions
	newBlock := NewBlock(validTransactions, lastBlock.Hash, difficulty)
	pow := NewProofOfWork(newBlock)
	nonce, hash, err := pow.Run()

	// Handle potential errors in the mining process
	if err != nil {
		fmt.Println("Error during Proof of Work:", err)
		return nil
	}
	newBlock.Hash = hash
	newBlock.Nonce = nonce
	newBlock.Index = len(bc.Blocks)

	// Validate the newly mined block before adding it to the chain
	if bc.IsValidNewBlock(newBlock, lastBlock) {
		bc.Blocks = append(bc.Blocks, newBlock)
		bc.clearMinedTransactions(validTransactions)
		return newBlock
	}
	return nil
}

// Validate whether a newly mined block is valid and follows the rules of the blockchain
func (bc *Blockchain) IsValidNewBlock(newBlock, previousBlock *Block) bool {
	
	// Check if the block index is consecutive
	if previousBlock.Index+1 != newBlock.Index {
		return false
	}

	// Check if the new block correctly references the previous block's hash
	if previousBlock.Hash != newBlock.PreviousHash {
		return false
	}

	// Validate the PoW
	pow := NewProofOfWork(newBlock)
	if !pow.Validate() {
		return false
	}

	// Recalculate the block's hash and compare
	if newBlock.calculateHash() != newBlock.Hash {
		return false
	}
	return true
}

// Validate the entire blockchain by checking each block's validity in order
func (bc *Blockchain) IsValidChain(blocks []*Block) bool {
	for i := 1; i < len(blocks); i++ {
		if !bc.IsValidNewBlock(blocks[i], blocks[i-1]) {
			return false
		}
	}
	return true
}

// Selects a proposer based on the amount of stake they hold
func (bc *Blockchain) SelectProposer() string {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	// Sum the total stake in the network
	totalStake := 0
	for _, stake := range bc.Stake {
		totalStake += stake
	}

	// If no stake is available, return an empty string
	if totalStake == 0 {
		return ""
	}

	// Select a proposer randomly, weighted by their stake
	weightedRandom := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomPoint := weightedRandom.Intn(totalStake)
	runningTotal := 0

	for address, stake := range bc.Stake {
		runningTotal += stake
		if runningTotal >= randomPoint {
			return address
		}
	}

	return ""
}

// Adds a new block using PoS consensus
func (bc *Blockchain) AddBlockPoS(transactions []*Transaction) *Block {
	bc.lock.Lock() // Lock the bc for writing
	defer bc.lock.Unlock() // Ensure unlocked after the operation

	// Select a proposer (the "miner" in PoS) based on their stake
	proposer := bc.SelectProposer()
	if proposer == "" { // If no proposer is found (maybe no one has any stake)
		fmt.Println("No stakes in the network, falling back to PoW")
		return bc.AddBlock(transactions)
	}

	// Get the last block in the chain
	lastBlock := bc.Blocks[len(bc.Blocks)-1]

	// Create a transaction to reward the propser (like a mining reward)
	minerRewardTx := &Transaction{
		Sender:    "system",			// System "creates" this reward
		Recipient: proposer,			// The propser get the reward
		Amount:    bc.GetBlockReward(), // Reward amount from the bc settings
		Fee:       0,					// No fee for this transaction
	}
	transactions = append([]*Transaction{minerRewardTx}, transactions...)

	// Create a new block with the given transactions
	newBlock := NewBlock(transactions, lastBlock.Hash, lastBlock.Difficulty)
	newBlock.Nonce = 0 // In PoS, nonce isn't really used, but it's part of the block struct

	// Validate new block before ading it to the chain
	if bc.IsValidNewBlock(newBlock, lastBlock) {
		bc.Blocks = append(bc.Blocks, newBlock)	// Add the block to the chain
		bc.clearMinedTransactions(transactions)	// Clear out these transactions from the mempool
		return newBlock
	}
	return nil // If block wasn't valid
}

// Upgrade the bc protocol to a new version
func (bc *Blockchain) UpgradeProtocol(version string) {
	bc.lock.Lock()	// Lock the bc for writing
	defer bc.lock.Unlock()	// Unlock after the operation is complete
	bc.ProtocolVersion = version	// Set the new protocol version
	fmt.Printf("Blockchain protocol upgraded to version %s\n", version)
}

func (bc *Blockchain) SetConsensusAlgorithm(algorithm string) {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	bc.ConsensusAlgorithm = algorithm
	fmt.Printf("Consensus algorithm set to %s\n", algorithm)
}

func (bc *Blockchain) SetMaxBlockSize(size int) {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	bc.MaxBlockSize = size
	fmt.Printf("Max block size set to %d bytes\n", size)
}

func (bc *Blockchain) SetBlockReward(reward int) {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	bc.blockReward = reward
	fmt.Printf("Block reward set to %d\n", reward)
}

func (bc *Blockchain) GetBlockReward() int {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	return bc.blockReward
}

// Selects the miner's address based on who has the most stake in the system
func (bc *Blockchain) SelectMinerAddress() string {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	// Find the address with the highest stake
	var highestStake int
	var minerAddress string

	for address, stake := range bc.Stake {
		if stake > highestStake {
			highestStake = stake
			minerAddress = address		// Upgrade to the address with the higest stake
		}
	}

	if minerAddress == "" {  // If no address with stake was found
		// Fallback to selecting a random address from the accounts
		for address := range bc.Accounts {
			minerAddress = address
			break
		}
	}

	return minerAddress
}

// Checks if a transaction is valid according to the bc's rules
func (bc *Blockchain) IsValidTransaction(tx *Transaction) bool {
	// Verify the transaction's signature using the sender's public key
	senderAccount := bc.Accounts[tx.Sender]
	if senderAccount == nil {
		return false // Transaction is invalid if the sender doesn't exist
	}

	// Verify the signature using the sender's public key
	if !tx.Verify(senderAccount.PublicKey) { 
		return false
	}

	// Check if the sender has enough balance to cover the amount and fee
	if senderAccount.Balance < tx.Amount+tx.Fee {
		return false
	}

	// Check the nonce to prevent replay attacks
	if tx.Nonce <= senderAccount.Nonce {
		return false
	}

	// Ensure the transaction fee meets the min required
	if tx.Fee < MinTransactionFee {
		return false
	}

	return true
}

// Removes transactions that have been successfully included in a block from the mempool 
func (bc *Blockchain) clearMinedTransactions(transactions []*Transaction) {
	// Lock the mempool to safely remove transactions
	bc.Mempool.lock.Lock()
	defer bc.Mempool.lock.Unlock()
	for _, tx := range transactions {
		bc.Mempool.RemoveTransaction(tx)  // Remove the transaction from the mempool
	}
}
