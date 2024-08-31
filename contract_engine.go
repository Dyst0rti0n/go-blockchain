package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Represents a basic smart contractr with its ID, code, creator, tiemstamp and State.
type SmartContract struct {
	ID        string
	Code      string
	Creator   string
	CreatedAt int64
	State     map[string]interface{}
}

// ContractEngine manages smart contracts. It's like a simple virtual machine for deploying and running contracts.
type ContractEngine struct {
	contracts map[string]*SmartContract
	lock      sync.RWMutex // Ensures thread-safe operations on contracts.
}

// NewContractEngine creates a new contract engine with an empty contract map.
func NewContractEngine() *ContractEngine {
	return &ContractEngine{
		contracts: make(map[string]*SmartContract),
	}
}

// DeployContract adds a new contract to the engine. It assigns a unique ID and initializes its state.
func (ce *ContractEngine) DeployContract(code, creator string) (string, error) {
	ce.lock.Lock()
	defer ce.lock.Unlock()

	contractID := generateContractID()  // Generate a unique ID for the contract
	contract := &SmartContract{
		ID:        contractID,
		Code:      code,
		Creator:   creator,
		CreatedAt: time.Now().Unix(),
		State:     make(map[string]interface{}),  // State starts empty
	}
	ce.contracts[contractID] = contract

	return contractID, nil
}

// ExecuteContract runs a specified method on a contract. If the method exists in the code, it performs the associated actions.
func (ce *ContractEngine) ExecuteContract(contractID, method string, params map[string]interface{}) (interface{}, error) {
	ce.lock.RLock() // Read lock for thread-safe access to the contract.
	defer ce.lock.RUnlock()

	contract, exists := ce.contracts[contractID]
	if !exists {
		return nil, fmt.Errorf("contract not found")
	}

	// Execute the contract code in a virtual environment (our simplistic interpreter).
	result, err := executeInVM(contract.Code, method, params, contract.State)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// executeInVM is a basic interpreter that processes simple contract code. It's not a real VM, just a toy example.
func executeInVM(code, method string, params map[string]interface{}, state map[string]interface{}) (interface{}, error) {
	lines := splitCodeIntoLines(code) // Break the code into lines.

	for _, line := range lines {
		parts := splitLine(line) // Split each line into parts (words or tokens).
		if len(parts) < 1 {
			continue
		}

		switch parts[0] { // Simple keyword-based command execution.
		case "SET":
			if len(parts) != 3 {
				return nil, errors.New("invalid SET command")
			}
			key := parts[1]
			value, exists := params[parts[2]]
			if !exists {
				value = parts[2] // Use literal value if not in params.
			}
			state[key] = value // Set the state key to the value.

		case "ADD":
			if len(parts) != 4 {
				return nil, errors.New("invalid ADD command")
			}
			key := parts[1]
			val1 := convertToInt(getValueFromParamsOrState(parts[2], params, state))
			val2 := convertToInt(getValueFromParamsOrState(parts[3], params, state))
			state[key] = val1 + val2 // Add two values and store in the state.

		case "CALL":
			if len(parts) != 2 {
				return nil, errors.New("invalid CALL command")
			}
			if parts[1] == method { // Execute the method if it matches the provided one.
				return state["RESULT"], nil
			}
		}
	}

	return nil, fmt.Errorf("method %s not found in contract", method)
}

// getValueFromParamsOrState fetches a value either from the parameters or the contract's state.
func getValueFromParamsOrState(key string, params, state map[string]interface{}) interface{} {
	if val, exists := params[key]; exists {
		return val
	}
	if val, exists := state[key]; exists {
		return val
	}
	return 0 // Default to 0 if the key is not found anywhere.
}

// splitCodeIntoLines breaks the contract code into individual lines.
func splitCodeIntoLines(code string) []string {
	return strings.Split(code, "\n")
}

// splitLine breaks a line of code into individual words or tokens.
func splitLine(line string) []string {
	return strings.Fields(line)
}

// convertToInt safely converts an interface value to an integer.
func convertToInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case string:
		result, err := strconv.Atoi(v)
		if err != nil {
			return 0
		}
		return result
	default:
		return 0
	}
}

// generateContractID creates a unique ID for a contract using the current timestamp.
func generateContractID() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))))
}