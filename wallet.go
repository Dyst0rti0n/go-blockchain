package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type WalletCLI struct {
	API *NodeAPIClient
}

func NewWalletCLI(api *NodeAPIClient) *WalletCLI {
	return &WalletCLI{API: api}
}

func (cli *WalletCLI) Run() {
	for {
		fmt.Println("1. Check Balance")
		fmt.Println("2. Send Transaction")
		fmt.Println("3. View Blockchain")
		fmt.Println("4. View Transaction")
		fmt.Println("5. Exit")
		fmt.Print("Enter choice: ")

		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			cli.handleCheckBalance()
		case 2:
			cli.handleSendTransaction()
		case 3:
			cli.handleViewBlockchain()
		case 4:
			cli.handleViewTransaction()
		case 5:
			return
		default:
			fmt.Println("Invalid choice")
		}
	}
}

func (cli *WalletCLI) handleCheckBalance() {
	fmt.Print("Enter address: ")
	var address string
	fmt.Scanln(&address)

	balance, err := cli.API.GetBalance(address)
	if err != nil {
		log.Printf("Failed to get balance: %v", err)
		return
	}

	fmt.Printf("Balance of %s: %d\n", address, balance)
}

func (cli *WalletCLI) handleSendTransaction() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter sender address: ")
	sender, _ := reader.ReadString('\n')
	sender = strings.TrimSpace(sender)

	fmt.Print("Enter recipient address: ")
	recipient, _ := reader.ReadString('\n')
	recipient = strings.TrimSpace(recipient)

	fmt.Print("Enter amount: ")
	amountStr, _ := reader.ReadString('\n')
	amount, _ := strconv.Atoi(strings.TrimSpace(amountStr))

	fmt.Print("Enter fee: ")
	feeStr, _ := reader.ReadString('\n')
	fee, _ := strconv.Atoi(strings.TrimSpace(feeStr))

	err := cli.API.SendTransaction(sender, recipient, amount, fee)
	if err != nil {
		log.Printf("Failed to send transaction: %v", err)
		return
	}

	fmt.Println("Transaction sent successfully!")
}

func (cli *WalletCLI) handleViewBlockchain() {
	blocks, err := cli.API.GetBlockchain()
	if err != nil {
		log.Printf("Failed to retrieve blockchain: %v", err)
		return
	}

	for _, block := range blocks {
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

func (cli *WalletCLI) handleViewTransaction() {
	fmt.Print("Enter transaction ID: ")
	var txID string
	fmt.Scanln(&txID)

	tx, err := cli.API.GetTransaction(txID)
	if err != nil {
		log.Printf("Failed to retrieve transaction: %v", err)
		return
	}

	fmt.Printf("Transaction ID: %s\n", tx.ID)
	fmt.Printf("Sender: %s\n", tx.Sender)
	fmt.Printf("Recipient: %s\n", tx.Recipient)
	fmt.Printf("Amount: %d\n", tx.Amount)
	fmt.Printf("Fee: %d\n", tx.Fee)
	fmt.Printf("Nonce: %d\n", tx.Nonce)
	fmt.Printf("Timestamp: %d\n", tx.Timestamp)
	fmt.Println()
}
