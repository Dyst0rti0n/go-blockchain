package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// Provides an HTTP API for interacting with the blockchain node.
type NodeAPI struct {
	Node *Node
}

// Client that interacts with the NodeAPI via HTTP requests.
type NodeAPIClient struct {
	BaseURL string
}

// Initialises a new NodeAPI with a given node.
func NewNodeAPI(node *Node) *NodeAPI {
	return &NodeAPI{Node: node}
}

// Creates a new client for communicating with the NodeAPI.
func NewNodeAPIClient(baseURL string) *NodeAPIClient {
	return &NodeAPIClient{BaseURL: baseURL}
}

// Start starts the HTTP API server on the specified port.
func (api *NodeAPI) Start(port string) error {
	http.HandleFunc("/balance", api.handleGetBalance)
	http.HandleFunc("/send", api.handleSendTransaction)
	http.HandleFunc("/blockchain", api.handleGetBlockchain)
	http.HandleFunc("/transaction", api.handleGetTransaction)
	log.Printf("API server running on port %s", port)
	return http.ListenAndServe(port, nil)
}

// Handles requests to get the balance of a specific address.
func (api *NodeAPI) handleGetBalance(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "Address is required", http.StatusBadRequest)
		return
	}
	balance := api.Node.Blockchain.UTXOSet.GetBalance(address)
	json.NewEncoder(w).Encode(map[string]int{"balance": balance})
}

// Handles requests to send a new transaction.
func (api *NodeAPI) handleSendTransaction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Sender    string `json:"sender"`
		Recipient string `json:"recipient"`
		Amount    int    `json:"amount"`
		Fee       int    `json:"fee"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx := &Transaction{
		Sender:    req.Sender,
		Recipient: req.Recipient,
		Amount:    req.Amount,
		Fee:       req.Fee,
	}

	// Sign the transaction with the node's private key
	err := tx.Sign(api.Node.PrivateKey)
	if err != nil {
		http.Error(w, "Failed to sign transaction", http.StatusInternalServerError)
		return
	}

	// Add the transaction to the mempool
	err = api.Node.Blockchain.Mempool.AddTransaction(tx, api.Node.Blockchain.Accounts, api.Node.Blockchain.UTXOSet)
	if err != nil {
		http.Error(w, "Failed to add transaction to the mempool", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "Transaction added to mempool"})
}

// Handles requests to get the entire blockchain.
func (api *NodeAPI) handleGetBlockchain(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(api.Node.Blockchain.Blocks)
}

// Handles requests to get a specific transaction by its ID.
func (api *NodeAPI) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	txID := r.URL.Query().Get("id")
	if txID == "" {
		http.Error(w, "Transaction ID is required", http.StatusBadRequest)
		return
	}

	tx := api.Node.Blockchain.Mempool.GetTransaction(txID)
	if tx == nil {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(tx)
}

// Sends a request to the NodeAPI to get the balance of a specific address.
func (api *NodeAPIClient) GetBalance(address string) (int, error) {
	resp, err := http.Get(fmt.Sprintf("%s/balance?address=%s", api.BaseURL, address))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	return result["balance"], nil
}

// Sends a transaction to the NodeAPI to be added to the blockchain.
func (api *NodeAPIClient) SendTransaction(sender, recipient string, amount, fee int) error {
	tx := map[string]interface{}{
		"sender":    sender,
		"recipient": recipient,
		"amount":    amount,
		"fee":       fee,
	}

	data, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	resp, err := http.Post(fmt.Sprintf("%s/send", api.BaseURL), "application/json", strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send transaction")
	}

	return nil
}

// Retrieves the entire blockchain from the NodeAPI.
func (api *NodeAPIClient) GetBlockchain() ([]*Block, error) {
	resp, err := http.Get(fmt.Sprintf("%s/blockchain", api.BaseURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var blocks []*Block
	if err := json.NewDecoder(resp.Body).Decode(&blocks); err != nil {
		return nil, err
	}

	return blocks, nil
}

// Retrieves a specific transaction by its ID from the NodeAPI.
func (api *NodeAPIClient) GetTransaction(txID string) (*Transaction, error) {
	resp, err := http.Get(fmt.Sprintf("%s/transaction?id=%s", api.BaseURL, txID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tx Transaction
	if err := json.NewDecoder(resp.Body).Decode(&tx); err != nil {
		return nil, err
	}

	return &tx, nil
}
