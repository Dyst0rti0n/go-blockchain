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

type ProposalStatus int

const (
	ProposalPending ProposalStatus = iota
	ProposalApproved
	ProposalRejected
	ProposalFailedQuorum
)

type Proposal struct {
	ID          string
	Description string
	Options     []string
	Votes       map[string]int
	Deadline    time.Time
	Status      ProposalStatus
	Executed    bool
	Category    string
	Quorum      int
}

type Governance struct {
	Proposals  map[string]*Proposal
	Votes      map[string]map[string]int
	Token      *Token
	Blockchain *Blockchain
	lock       sync.Mutex
}

func NewGovernance(token *Token, blockchain *Blockchain) *Governance {
	return &Governance{
		Proposals:  make(map[string]*Proposal),
		Votes:      make(map[string]map[string]int),
		Token:      token,
		Blockchain: blockchain,
	}
}

func (g *Governance) CreateProposal(description, category string, options []string, duration time.Duration, quorum int) (string, error) {
	g.lock.Lock()
	defer g.lock.Unlock()

	proposalID := fmt.Sprintf("%x", sha256.Sum256([]byte(description+time.Now().String())))
	deadline := time.Now().Add(duration)

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
	proposal.Votes[proposal.Options[optionIndex]] += voterBalance

	g.logEvent(fmt.Sprintf("Vote cast on proposal %s by %s", proposalID, voterAddress))
	return nil
}

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

	totalVotes := 0
	for _, votes := range proposal.Votes {
		totalVotes += votes
	}

	if totalVotes < proposal.Quorum {
		proposal.Status = ProposalFailedQuorum
		return "", errors.New("quorum not met")
	}

	var winningOption string
	maxVotes := -1

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

func (g *Governance) executeNetworkUpgrade(proposal *Proposal) {
	g.logEvent(fmt.Sprintf("Executing network upgrade: %s", proposal.Description))

	if len(proposal.Options) == 0 {
		g.logEvent("Network upgrade failed: No options provided in proposal")
		return
	}

	upgradeAction := proposal.Options[0]

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

func (g *Governance) logEvent(event string) {
	fmt.Printf("Governance Event: %s\n", event)
}

