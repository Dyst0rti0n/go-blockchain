package main

import (
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

var privateKey *ecdsa.PrivateKey
var publicKey *ecdsa.PublicKey

func init() {
	privKey, pubKey, err := GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}
	privateKey = privKey
	publicKey = pubKey
}

func main() {
	nodeAddress := flag.String("node", "localhost:8080", "Node address")
	knownPeers := flag.String("peers", "", "Comma-separated list of known peers")
	apiPort := flag.String("api", ":8081", "API server port")
	mode := flag.String("mode", "full", "Node mode (full, light, api)")
	flag.Parse()

	blockchain := NewBlockchain()
	blockchain.UTXOSet = NewUTXOSet()
	blockchain.Accounts = make(map[string]*Account)
	blockchain.Mempool = NewMempool()
	database := NewInMemoryDatabase()
	gamification := NewGamification(database)

	node := NewNode(*nodeAddress, blockchain, privateKey)

	initializeBlockchainWithGenesis(blockchain)

	if *mode == "api" {
		api := NewNodeAPI(node)
		go func() {
			log.Fatal(api.Start(*apiPort))
		}()
	}

	cli := NewWalletCLI(NewNodeAPIClient(fmt.Sprintf("http://localhost%s", *apiPort)))
	cli.Run()

	// Discover and connect to known peers
	if *knownPeers != "" {
		node.DiscoverPeers(parsePeers(*knownPeers))
	}

	switch *mode {
	case "full":
		go func() {
			log.Fatal(node.Start())
		}()
	case "api":
		api := NewNodeAPI(node)
		go func() {
			log.Fatal(api.Start(*apiPort))
		}()
	case "light":
		fmt.Println("Light mode currently under development.")
		// Implement light node functionality
	default:
		fmt.Println("Invalid mode specified.")
		os.Exit(1)
	}

	cliLoop(blockchain, node, gamification)
}


func cliLoop(bc *Blockchain, node *Node, gamification *Gamification) {
	for {
		fmt.Println("1. Create Transaction")
		fmt.Println("2. Mine Block")
		fmt.Println("3. Print Blockchain")
		fmt.Println("4. Deploy Smart Contract")
		fmt.Println("5. Execute Smart Contract")
		fmt.Println("6. Register DID")
		fmt.Println("7. Authenticate DID")
		fmt.Println("8. Switch Consensus Algorithm")
		fmt.Println("9. Exit")
		fmt.Print("Enter choice: ")

		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			handleCreateTransaction(bc.Mempool, bc)
		case 2:
			handleMineBlock(bc, bc.Mempool, gamification, bc.UTXOSet)
		case 3:
			handlePrintBlockchain(bc)
		case 4:
			handleDeploySmartContract(bc)
		case 5:
			handleExecuteSmartContract(bc)
		case 6:
			handleRegisterDID(bc)
		case 7:
			handleAuthenticateDID(bc)
		case 8:
			handleSwitchConsensus(bc)
		case 9:
			return
		default:
			fmt.Println("Invalid choice")
		}
	}
}

func handleSwitchConsensus(bc *Blockchain) {
	fmt.Println("Available consensus algorithms: PoW, PoS")
	fmt.Print("Enter the new consensus algorithm: ")
	var algo string
	fmt.Scanln(&algo)

	bc.SetConsensusAlgorithm(algo)
	fmt.Printf("Switched to %s consensus algorithm.\n", algo)
}

func handleCreateTransaction(tp *Mempool, bc *Blockchain) {
	var sender, recipient string
	var amount, fee int
	fmt.Print("Enter sender: ")
	fmt.Scanln(&sender)
	fmt.Print("Enter recipient: ")
	fmt.Scanln(&recipient)
	fmt.Print("Enter amount: ")
	fmt.Scanln(&amount)
	fmt.Print("Enter fee: ")
	fmt.Scanln(&fee)

	// Ensure sender has enough balance
	senderBalance := bc.UTXOSet.GetBalance(sender)
	if senderBalance < amount+fee {
		fmt.Println("Insufficient balance.")
		return
	}

	tx := &Transaction{Sender: sender, Recipient: recipient, Amount: amount, Fee: fee}
	err := tx.Sign(privateKey)
	if err != nil {
		fmt.Println("Failed to sign transaction:", err)
		return
	}

	if !tx.Verify(publicKey) {
		fmt.Println("Failed to verify transaction signature.")
		return
	}

	err = tx.Validate(bc.Accounts, bc.UTXOSet)
	if err != nil {
		fmt.Println("Transaction validation failed:", err)
		return
	}

	err = tp.AddTransaction(tx, bc.Accounts, bc.UTXOSet)
	if err != nil {
		fmt.Println("Failed to add transaction to the mempool:", err)
		return
	}

	fmt.Println("Transaction created and added to the mempool.")
}

func handleMineBlock(bc *Blockchain, tp *Mempool, gamification *Gamification, utxoSet *UTXOSet) {
	minerAddress := "miner-address" // Replace with the actual miner address

	// Initialize miner's address in UTXO set if not already present
	if !utxoSet.HasUTXO(minerAddress) {
		utxoSet.AddUTXO(UTXO{
			Owner:  minerAddress,
			Amount: 0,
			TxID:   "genesis",
			Index:  0,
		})
	}

	// Enforce cooldown period
	user, _ := gamification.loadOrCreateUser(minerAddress) // Load or create the user object
	err := gamification.EnforceCooldown(user, "mining")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Detect suspicious patterns
	err = gamification.DetectSuspiciousPatterns(user)
	if err != nil {
		fmt.Println(err)
		return
	}

	transactions := tp.GetTransactions()
	newBlock := bc.AddBlock(transactions)
	if newBlock == nil {
		fmt.Println("Failed to mine block.")
		return
	}

	tp.Clear() // Clear the mempool using the correct method

	// Reward the miner with points for successful block mining
	gamification.RewardUser(minerAddress, 100, "mining")

	// Optionally distribute fees and rewards among participants
	for _, tx := range transactions {
		tx.DistributeFees(utxoSet, minerAddress)
	}

	fmt.Println("Block mined successfully!")
}

func handleDeploySmartContract(bc *Blockchain) {
	var code string
	fmt.Print("Enter smart contract code: ")
	fmt.Scanln(&code)

	contractID, err := bc.ContractEngine.DeployContract(code, "user-address") // Replace with the actual user address
	if err != nil {
		fmt.Println("Failed to deploy smart contract:", err)
		return
	}

	fmt.Printf("Smart contract deployed with ID: %s\n", contractID)
}

func handleExecuteSmartContract(bc *Blockchain) {
	var contractID, method string
	fmt.Print("Enter smart contract ID: ")
	fmt.Scanln(&contractID)
	fmt.Print("Enter method name: ")
	fmt.Scanln(&method)

	params := make(map[string]interface{})
	// Collect parameters here if needed

	result, err := bc.ContractEngine.ExecuteContract(contractID, method, params)
	if err != nil {
		fmt.Println("Failed to execute smart contract:", err)
		return
	}

	fmt.Printf("Smart contract executed. Result: %v\n", result)
}

func handleRegisterDID(bc *Blockchain) {
	var publicKey string
	fmt.Print("Enter public key: ")
	fmt.Scanln(&publicKey)

	attributes := make(map[string]string)
	// Collect additional attributes if needed

	didID, err := bc.DIDRegistry.RegisterDID("user-address", publicKey, attributes) // Replace with the actual user address
	if err != nil {
		fmt.Println("Failed to register DID:", err)
		return
	}

	fmt.Printf("DID registered with ID: %s\n", didID)
}

func handleAuthenticateDID(bc *Blockchain) {
	var didID, signature, message string
	fmt.Print("Enter DID ID: ")
	fmt.Scanln(&didID)
	fmt.Print("Enter signature: ")
	fmt.Scanln(&signature)
	fmt.Print("Enter message: ")
	fmt.Scanln(&message)

	isValid, err := bc.DIDRegistry.AuthenticateDID(didID, signature, message)
	if err != nil {
		fmt.Println("Failed to authenticate DID:", err)
		return
	}

	if isValid {
		fmt.Println("DID authentication successful.")
	} else {
		fmt.Println("DID authentication failed.")
	}
}

func handlePrintBlockchain(bc *Blockchain) {
	for _, block := range bc.Blocks {
		fmt.Printf("Index: %d\n", block.Index)
		fmt.Printf("Timestamp: %d\n", block.Timestamp)
		fmt.Printf("Previous Hash: %s\n", block.PreviousHash)
		fmt.Printf("Hash: %s\n", block.Hash)
		fmt.Printf("Transactions: %v\n", block.Transactions)
		fmt.Printf("Nonce: %d\n", block.Nonce)
		fmt.Printf("Difficulty: %d\n", block.Difficulty)
		fmt.Println()
	}
}

func parsePeers(peers string) []string {
	return strings.Split(peers, ",")
}

func initializeBlockchainWithGenesis(blockchain *Blockchain) {
	// Assign some initial UTXOs to users for testing
	genesisTransaction := &Transaction{
		Sender:    "system",
		Recipient: "bob",
		Amount:    100,
		Fee:       0,
	}

	// Add this transaction to the UTXO set
	blockchain.UTXOSet.AddUTXO(UTXO{
		Owner:  "bob",
		Amount: 100,
		TxID:   genesisTransaction.Hash(),
		Index:  0,
	})

	// Mine the genesis block
	blockchain.AddBlock([]*Transaction{genesisTransaction})
}