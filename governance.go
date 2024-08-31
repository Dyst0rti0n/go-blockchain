package main

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// ProposalStatus represents the different states a proposal can be in during the governance process.
type ProposalStatus int

const (
	ProposalPending ProposalStatus = iota   // The proposal is currently open for voting.
	ProposalApproved                        // The proposal has been approved by voters.
	ProposalRejected                        // The proposal has been rejected by voters.
	ProposalFailedQuorum                    // The proposal did not meet the required quorum.
)

// Proposal represents a governance proposal that members of the network can vote on.
type Proposal struct {
	ID          string
	Description string
	Options     []string
	Votes       map[string]int
	Deadline    time.Time
	Status      ProposalStatus
	Executed    bool
	Category    string
	Quorum      int  // The minimum number of votes required for the proposal to be valid.
}

// Governance handles the creation, voting, tallying, and execution of proposals in a decentralized system.
type Governance struct {
	Proposals  map[string]*Proposal
	Votes      map[string]map[string]int
	Token      *Token         // Token represents the governance token used for voting.
	Blockchain *Blockchain    // Blockchain represents the underlying blockchain where the governance operates.
	lock       sync.Mutex     // Mutex to ensure thread-safe operations.
}

// NewGovernance creates a new instance of the Governance system.
func NewGovernance(token *Token, blockchain *Blockchain) *Governance {
	return &Governance{
		Proposals:  make(map[string]*Proposal),
		Votes:      make(map[string]map[string]int),
		Token:      token,
		Blockchain: blockchain,
	}
}

// CreateProposal allows users to create a new governance proposal.
// The proposal includes a description, category, options, voting duration, and quorum requirement.
func (g *Governance) CreateProposal(description, category string, options []string, duration time.Duration, quorum int) (string, error) {
	g.lock.Lock()
	defer g.lock.Unlock()

	// Generate a unique proposal ID using a hash of the description and current time.
	proposalID := fmt.Sprintf("%x", sha256.Sum256([]byte(description+time.Now().String())))
	deadline := time.Now().Add(duration) // Set the deadline for voting.

	// Initialize the proposal with the provided details.
	proposal := &Proposal{
		ID:          proposalID,
		Description: description,
		Options:     options,
		Votes:       make(map[string]int),
		Deadline:    deadline,
		Status:      ProposalPending,
		Executed:    false,
		Category:    category,
		Quorum:      quorum,
	}

	g.Proposals[proposalID] = proposal
	g.logEvent(fmt.Sprintf("Proposal created: %s, Category: %s", description, category))
	return proposalID, nil
}

// Vote allows a user to cast their vote on a specific proposal.
// The user's vote is weighted based on the number of governance tokens they hold.
func (g *Governance) Vote(proposalID, voterAddress string, optionIndex int, privateKey *ecdsa.PrivateKey) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	proposal, exists := g.Proposals[proposalID]
	if !exists {
		return errors.New("proposal not found")
	}

	if time.Now().After(proposal.Deadline) {
		return errors.New("voting period has ended")
	}

	voterBalance := g.Token.BalanceOf(voterAddress)
	if voterBalance <= 0 {
		return errors.New("voter has no tokens")
	}

	if _, voted := g.Votes[proposalID][voterAddress]; voted {
		return errors.New("voter has already voted")
	}

	if g.Votes[proposalID] == nil {
		g.Votes[proposalID] = make(map[string]int)
	}
	g.Votes[proposalID][voterAddress] = optionIndex
	proposal.Votes[proposal.Options[optionIndex]] += voterBalance // Vote is weighted by token balance.

	g.logEvent(fmt.Sprintf("Vote cast on proposal %s by %s", proposalID, voterAddress))
	return nil
}

// TallyVotes counts the votes for a specific proposal and determines the winning option.
// If the proposal meets the quorum, it is approved, otherwise, it fails due to insufficient participation.
func (g *Governance) TallyVotes(proposalID string) (string, error) {
	g.lock.Lock()
	defer g.lock.Unlock()

	proposal, exists := g.Proposals[proposalID]
	if !exists {
		return "", errors.New("proposal not found")
	}

	if time.Now().Before(proposal.Deadline) {
		return "", errors.New("voting period has not ended")
	}

	// Calculate the total number of votes cast.
	totalVotes := 0
	for _, votes := range proposal.Votes {
		totalVotes += votes
	}

	// Check if the quorum is met.
	if totalVotes < proposal.Quorum {
		proposal.Status = ProposalFailedQuorum
		return "", errors.New("quorum not met")
	}

	var winningOption string
	maxVotes := -1

	// Determine which option received the most votes.
	for option, votes := range proposal.Votes {
		if votes > maxVotes {
			winningOption = option
			maxVotes = votes
		}
	}

	proposal.Status = ProposalApproved
	g.logEvent(fmt.Sprintf("Proposal %s approved with option %s", proposalID, winningOption))
	return winningOption, nil
}

// ExecuteProposal carries out the actions of an approved proposal based on its category and options.
func (g *Governance) ExecuteProposal(proposalID string) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	proposal, exists := g.Proposals[proposalID]
	if !exists {
		return errors.New("proposal not found")
	}

	if proposal.Executed {
		return errors.New("proposal already executed")
	}

	if proposal.Status != ProposalApproved {
		return errors.New("proposal not approved")
	}

	// Execute the proposal based on its category.
	switch proposal.Category {
	case "network-upgrade":
		g.executeNetworkUpgrade(proposal)

	case "block-reward":
		g.executeBlockRewardChange(proposal)

	default:
		return errors.New("unknown proposal action")
	}

	proposal.Executed = true
	g.logEvent(fmt.Sprintf("Proposal %s executed", proposalID))
	return nil
}

// executeNetworkUpgrade handles the implementation of a network upgrade proposal.
// This can involve upgrading the protocol, changing the consensus algorithm, or increasing block size.
func (g *Governance) executeNetworkUpgrade(proposal *Proposal) {
	g.logEvent(fmt.Sprintf("Executing network upgrade: %s", proposal.Description))

	if len(proposal.Options) == 0 {
		g.logEvent("Network upgrade failed: No options provided in proposal")
		return
	}

	upgradeAction := proposal.Options[0]

	// Perform the network upgrade based on the winning option.
	switch upgradeAction {
	case "Upgrade to v2.0":
		g.Blockchain.UpgradeProtocol("v2.0")
		g.logEvent("Blockchain upgraded to protocol version 2.0")

	case "Enable new consensus algorithm":
		g.Blockchain.SetConsensusAlgorithm("PoS")
		g.logEvent("Consensus algorithm switched to Proof of Stake (PoS)")

	case "Increase block size":
		g.Blockchain.SetMaxBlockSize(2_000_000) // Example increase to 2 MB
		g.logEvent("Max block size increased to 2 MB")

	default:
		g.logEvent(fmt.Sprintf("Unknown network upgrade action: %s", upgradeAction))
		return
	}

	g.logEvent(fmt.Sprintf("Network upgrade completed: %s", proposal.Description))
}

// executeBlockRewardChange implements a proposal to change the block reward.
// This involves updating the block reward parameter in the blockchain based on the winning option.
func (g *Governance) executeBlockRewardChange(proposal *Proposal) {
	winningOption, err := g.TallyVotes(proposal.ID)
	if err != nil {
		fmt.Println("Error tallying votes:", err)
		return
	}

	newReward, err := strconv.Atoi(winningOption)
	if err != nil {
		fmt.Println("Invalid block reward value:", winningOption)
		return
	}

	g.Blockchain.SetBlockReward(newReward)
	fmt.Printf("Block reward changed to %d based on proposal %s\n", newReward, proposal.ID)
}

// logEvent logs events related to the governance process for transparency and auditing purposes.
func (g *Governance) logEvent(event string) {
	fmt.Printf("Governance Event: %s\n", event)
}
