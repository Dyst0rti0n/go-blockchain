package main

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

const (
	RateLimitWindow      = 10 * time.Second  // Time window for rate limiting peer requests.
	MaxRequestsPerWindow = 100               // Maximum requests allowed within the rate limit window.
	MaxConnectionRetries = 3                 // Maximum retries for peer connections.
	RetryDelay           = 2 * time.Second   // Delay between connection retries.
)

type MessageType int

const (
	MessageTypeNewBlock MessageType = iota     // New block message type.
	MessageTypeTransaction                     // Transaction message type.
	MessageTypeRequestBlockchain               // Request for the entire blockchain.
	MessageTypeResponseBlockchain              // Response containing the entire blockchain.
	MessageTypeNewPeer                         // Message indicating a new peer connection.
)

type Message struct {
	Type    MessageType   // Type of the message.
	Payload []byte        // Content of the message.
}

type Node struct {
	Address          string            // The node's address.
	Blockchain       *Blockchain       // The blockchain instance associated with the node.
	Peers            map[string]bool   // A map of known peer addresses.
	lock             sync.RWMutex      // A read-write lock for thread-safe operations.
	requestCounts    map[string]int    // Counts the number of requests per peer.
	lastRequestTimes map[string]time.Time // Tracks the last request time per peer.
	messageQueue     chan Message      // A queue for processing incoming messages.
	PrivateKey       *ecdsa.PrivateKey // The node's private key for signing transactions.
}

func NewNode(address string, blockchain *Blockchain, privateKey *ecdsa.PrivateKey) *Node {
	return &Node{
		Address:          address,
		Blockchain:       blockchain,
		Peers:            make(map[string]bool),
		requestCounts:    make(map[string]int),
		lastRequestTimes: make(map[string]time.Time),
		messageQueue:     make(chan Message, 100),
		PrivateKey:       privateKey,
	}
}

// Load the TLS configuration for secure connections, including server and CA certificates.
func loadTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		return nil, err
	}
	caCert, err := os.ReadFile("ca.crt")
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}, nil
}

// Start the node's main operations, including listening for connections and processing messages.
func (n *Node) Start() error {
	tlsConfig, err := loadTLSConfig()
	if err != nil {
		return err
	}

	ln, err := tls.Listen("tcp", n.Address, tlsConfig)
	if err != nil {
		return err
	}
	defer ln.Close()

	fmt.Printf("Secure Node started at %s\n", n.Address)

	go n.processMessageQueue()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go n.handleConnection(conn)
	}
}

// Handle incoming connections from peers, including rate limiting and message decoding.
func (n *Node) handleConnection(conn net.Conn) {
	defer conn.Close()

	peerAddr := conn.RemoteAddr().String()
	if !n.rateLimit(peerAddr) {
		log.Printf("Rate limit exceeded for peer: %s", peerAddr)
		return
	}

	var msg Message
	decoder := json.NewDecoder(conn)
	err := decoder.Decode(&msg)
	if err != nil {
		log.Printf("Failed to decode message: %v", err)
		return
	}

	switch msg.Type {
	case MessageTypeRequestBlockchain:
		n.handleRequestBlockchain(conn)
	default:
		n.messageQueue <- msg
	}
}

// Implement rate limiting to prevent peers from overwhelming the node with requests.
func (n *Node) rateLimit(peerAddr string) bool {
	now := time.Now()
	n.lock.Lock()
	defer n.lock.Unlock()

	if lastRequestTime, exists := n.lastRequestTimes[peerAddr]; exists {
		if now.Sub(lastRequestTime) > RateLimitWindow {
			n.requestCounts[peerAddr] = 0
		}
	}

	n.lastRequestTimes[peerAddr] = now
	n.requestCounts[peerAddr]++

	return n.requestCounts[peerAddr] <= MaxRequestsPerWindow
}

// Continuously process messages from the message queue, dispatching them to the appropriate handlers.
func (n *Node) processMessageQueue() {
	for msg := range n.messageQueue {
		switch msg.Type {
		case MessageTypeNewBlock:
			n.handleNewBlock(msg.Payload)
		case MessageTypeTransaction:
			n.handleTransaction(msg.Payload)
		case MessageTypeResponseBlockchain:
			n.handleResponseBlockchain(msg.Payload)
		case MessageTypeNewPeer:
			n.handleNewPeer(msg.Payload)
		}
	}
}

// Handle the reception of a new block, validate it, and propagate it to peers.
func (n *Node) handleNewBlock(payload []byte) {
	var block Block
	err := json.Unmarshal(payload, &block)
	if err != nil {
		log.Printf("Failed to unmarshal block: %v", err)
		return
	}
	if n.Blockchain.IsValidNewBlock(&block, n.Blockchain.Blocks[len(n.Blockchain.Blocks)-1]) {
		n.Blockchain.lock.Lock()
		n.Blockchain.Blocks = append(n.Blockchain.Blocks, &block)
		n.Blockchain.lock.Unlock()
		n.broadcastToPeers(MessageTypeNewBlock, payload)
	}
}

// Handle the reception of a transaction, validate it, and propagate it to peers.
func (n *Node) handleTransaction(payload []byte) {
	var tx Transaction
	err := json.Unmarshal(payload, &tx)
	if err != nil {
		log.Printf("Failed to unmarshal transaction: %v", err)
		return
	}
	if err := n.Blockchain.Mempool.AddTransaction(&tx, n.Blockchain.Accounts, n.Blockchain.UTXOSet); err != nil {
		log.Printf("Failed to add transaction to mempool: %v", err)
		return
	}
	n.broadcastToPeers(MessageTypeTransaction, payload)
}

// Respond to requests for the entire blockchain by sending the blockchain data to the requesting peer.
func (n *Node) handleRequestBlockchain(conn net.Conn) {
	n.Blockchain.lock.RLock()
	defer n.Blockchain.lock.RUnlock()

	data, err := json.Marshal(n.Blockchain)
	if err != nil {
		log.Printf("Failed to marshal blockchain: %v", err)
		return
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(data); err != nil {
		log.Printf("Failed to send blockchain data: %v", err)
	}
}

// Handle the reception of a blockchain from a peer, and update the node's blockchain if the received one is valid and longer.
func (n *Node) handleResponseBlockchain(payload []byte) {
	var receivedBlockchain Blockchain
	err := json.Unmarshal(payload, &receivedBlockchain)
	if err != nil {
		log.Printf("Failed to unmarshal blockchain response: %v", err)
		return
	}
	if len(receivedBlockchain.Blocks) > len(n.Blockchain.Blocks) && n.Blockchain.IsValidChain(receivedBlockchain.Blocks) {
		n.Blockchain.lock.Lock()
		n.Blockchain.Blocks = receivedBlockchain.Blocks
		n.Blockchain.lock.Unlock()
	}
}

// Handle the addition of a new peer to the node's list of known peers and attempt to establish a connection.
func (n *Node) handleNewPeer(payload []byte) {
	var peerAddress string
	err := json.Unmarshal(payload, &peerAddress)
	if err != nil {
		log.Printf("Failed to unmarshal new peer address: %v", err)
		return
	}
	n.lock.Lock()
	defer n.lock.Unlock()
	if !n.Peers[peerAddress] {
		n.Peers[peerAddress] = true
		go n.connectToPeer(peerAddress)
	}
}

// Attempt to establish a secure connection to a peer and notify them of the new connection.
func (n *Node) connectToPeer(address string) {
	for i := 0; i < MaxConnectionRetries; i++ {
		tlsConfig, err := loadTLSConfig()
		if err != nil {
			log.Printf("Failed to load TLS config: %v", err)
			return
		}

		conn, err := tls.Dial("tcp", address, tlsConfig)
		if err != nil {
			log.Printf("Failed to connect to peer %s: %v", address, err)
			time.Sleep(RetryDelay)
			continue
		}
		defer conn.Close()

		msg := Message{Type: MessageTypeNewPeer, Payload: []byte(n.Address)}
		encoder := json.NewEncoder(conn)
		err = encoder.Encode(msg)
		if err != nil {
			log.Printf("Failed to send new peer message to %s: %v", address, err)
			time.Sleep(RetryDelay)
			continue
		}

		n.lock.Lock()
		n.Peers[address] = true
		n.lock.Unlock()
		break
	}
}

// Broadcast a message to all known peers in the network.
func (n *Node) broadcastToPeers(msgType MessageType, payload []byte) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	for peer := range n.Peers {
		go func(peer string) {
			for i := 0; i < MaxConnectionRetries; i++ {
				tlsConfig, err := loadTLSConfig()
				if err != nil {
					log.Printf("Failed to load TLS config for peer %s: %v", peer, err)
					return
				}

				conn, err := tls.Dial("tcp", peer, tlsConfig)
				if err != nil {
					log.Printf("Failed to connect to peer %s: %v", peer, err)
					time.Sleep(RetryDelay)
					continue
				}
				defer conn.Close()

				msg := Message{Type: msgType, Payload: payload}
				encoder := json.NewEncoder(conn)
				err = encoder.Encode(msg)
				if err != nil {
					log.Printf("Failed to send message to peer %s: %v", peer, err)
					time.Sleep(RetryDelay)
					continue
				}
				break
			}
		}(peer)
	}
}

// Attempt to connect to a list of known peers, establishing connections with those that are
func (n *Node) DiscoverPeers(knownPeers []string) {
	for _, peer := range knownPeers {
		if peer != n.Address {
			go n.connectToPeer(peer)
		}
	}
}