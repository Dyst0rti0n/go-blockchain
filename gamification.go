package main

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Badge represents an achievement earned by the user.
// Badges can be thought of as rewards or honors for meeting certain criteria within the system.
type Badge struct {
	Name        string
	Description string
	Criteria    func(user *User) bool // Criteria function to dynamically assign badges
}

// User represents a participant in the blockchain network.
// Each user has an address, points, level, badges, and a record of their last activity.
type User struct {
	Address    string
	Points     int
	Level      int
	Badges     map[string]Badge
	LastActive time.Time
	RewardLog  map[string]time.Time // Logs when the user last received rewards for specific activities
}

// Gamification handles user rewards, levels, and badges.
// This system encourages user engagement by rewarding points, assigning badges, and promoting users through levels.
type Gamification struct {
	Users       map[string]*User
	Badges      map[string]Badge
	Leaderboard *Leaderboard
	lock        sync.Mutex
	Levels      []int // Thresholds for leveling up
	Cooldowns   map[string]time.Duration
	db          *InMemoryDatabase
}

// Leaderboard maintains the ranking of users based on their points.
// The leaderboard showcases the top performers in the network.
type Leaderboard struct {
	Users []*User
	lock  sync.Mutex
}

// NewGamification initializes a new Gamification system with a database.
// This sets up the basic structure for managing user points, levels, badges, and the leaderboard.
func NewGamification(db *InMemoryDatabase) *Gamification {
	return &Gamification{
		Users:       make(map[string]*User),
		Badges:      make(map[string]Badge),
		Leaderboard: &Leaderboard{Users: []*User{}},
		Levels: []int{
			100, 500, 1000, 2000, 5000, 10000, // Define level thresholds here
		},
		Cooldowns: map[string]time.Duration{
			"mining":    10 * time.Minute,
			"voting":    5 * time.Minute,
			"liquidity": 30 * time.Minute,
		},
		db: db,
	}
}

// RewardUser rewards a user with points and handles leveling and badges.
// This function manages point allocation, level progression, and badge assignments.
func (g *Gamification) RewardUser(address string, points int, activity string) error {
	g.lock.Lock()
	defer g.lock.Unlock()

	user, err := g.loadOrCreateUser(address)
	if err != nil {
		return err
	}

	if err := g.EnforceCooldown(user, activity); err != nil {
		return err
	}

	user.Points += points
	user.LastActive = time.Now()
	user.RewardLog[activity] = time.Now()

	g.db.Set(user.Address, user) // Save user to the database

	g.checkLevelUp(user)
	g.assignBadges(user)

	g.Leaderboard.UpdateLeaderboard(user)
	return nil
}

// loadOrCreateUser retrieves a user from memory or creates a new one if they don't exist.
func (g *Gamification) loadOrCreateUser(address string) (*User, error) {
	if user, exists := g.Users[address]; exists {
		return user, nil
	}

	if value, exists := g.db.Get(address); exists {
		user := value.(*User)
		g.Users[address] = user
		return user, nil
	}

	// Initialize a new user if they don't exist yet
	user := &User{
		Address:    address,
		Points:     0,
		Level:      1,
		Badges:     make(map[string]Badge),
		LastActive: time.Now(),
		RewardLog:  make(map[string]time.Time),
	}
	g.Users[address] = user
	return user, nil
}

// checkLevelUp checks if the user has enough points to level up.
// If the user's points meet or exceed the threshold for the next level, they are leveled up.
func (g *Gamification) checkLevelUp(user *User) {
	for i, threshold := range g.Levels {
		if user.Points >= threshold && user.Level <= i+1 {
			user.Level = i + 2 // Levels are 1-based, thresholds are 0-based
			fmt.Printf("User %s leveled up to level %d!\n", user.Address, user.Level)
		}
	}
}

// assignBadges checks if the user qualifies for any badges and assigns them if they do.
// This function dynamically awards badges based on user activities.
func (g *Gamification) assignBadges(user *User) {
	for _, badge := range g.Badges {
		if _, hasBadge := user.Badges[badge.Name]; !hasBadge && badge.Criteria(user) {
			user.Badges[badge.Name] = badge
			fmt.Printf("User %s earned badge: %s\n", user.Address, badge.Name)
		}
	}
}

// RewardFullNode rewards users based on full node uptime.
// This incentivizes users to maintain full nodes by awarding points based on uptime.
func (g *Gamification) RewardFullNode(address string, uptime time.Duration) error {
	points := int(uptime.Hours()) * 10 // Example: 10 points per hour of uptime
	return g.RewardUser(address, points, "full-node")
}

// RewardLiquidityProvider rewards users for providing liquidity.
// This incentivizes users to add liquidity to the network by awarding points based on the amount provided.
func (g *Gamification) RewardLiquidityProvider(address string, liquidityProvided int) error {
	points := liquidityProvided / 100 // Example: 1 point per 100 units of liquidity
	return g.RewardUser(address, points, "liquidity")
}

// RewardTransactionFees rewards users for contributing transaction fees.
// This incentivizes users who contribute to transaction fees by awarding points based on the fees paid.
func (g *Gamification) RewardTransactionFees(address string, feesContributed int) error {
	points := feesContributed / 10 // Example: 1 point per 10 units of fees
	return g.RewardUser(address, points, "transaction-fees")
}

// DetectSuspiciousPatterns detects suspicious activity based on the frequency of user actions.
// This is a simple anti-cheating mechanism to prevent users from abusing the reward system.
func (g *Gamification) DetectSuspiciousPatterns(user *User) error {
	activityFrequency := time.Since(user.LastActive)
	if activityFrequency < 1*time.Minute {
		return errors.New("suspicious activity detected: too frequent actions")
	}
	return nil
}

// EnforceCooldown ensures that a user cannot perform the same activity too frequently.
// This prevents abuse of the reward system by enforcing a cooldown period between actions.
func (g *Gamification) EnforceCooldown(user *User, activity string) error {
	if cooldown, ok := g.Cooldowns[activity]; ok {
		if time.Since(user.RewardLog[activity]) < cooldown {
			return fmt.Errorf("cooldown period for %s not met", activity)
		}
	}
	return nil
}

// Leaderboard management
// UpdateLeaderboard updates the user's position on the leaderboard based on their points.
func (lb *Leaderboard) UpdateLeaderboard(user *User) {
	lb.lock.Lock()
	defer lb.lock.Unlock()

	// Find and update the user's position
	for i, u := range lb.Users {
		if u.Address == user.Address {
			lb.Users[i] = user
			sort.Slice(lb.Users, func(i, j int) bool {
				return lb.Users[i].Points > lb.Users[j].Points
			})
			return
		}
	}
	// If the user is not on the leaderboard, add them
	lb.Users = append(lb.Users, user)
	sort.Slice(lb.Users, func(i, j int) bool {
		return lb.Users[i].Points > lb.Users[j].Points
	})
}

// DisplayTopUsers prints out the top N users on the leaderboard.
func (lb *Leaderboard) DisplayTopUsers(n int) {
	lb.lock.Lock()
	defer lb.lock.Unlock()

	fmt.Println("Top Users:")
	for i := 0; i < n && i < len(lb.Users); i++ {
		user := lb.Users[i]
		fmt.Printf("%d. %s - %d Points\n", i+1, user.Address, user.Points)
	}
}

// InMemoryDatabase is a simple in-memory key-value store for user data.
// This is a basic way to store and retrieve user data without using an external database.
type InMemoryDatabase struct {
	data map[string]interface{}
	lock sync.RWMutex
}

// NewInMemoryDatabase initializes a new in-memory database.
// This creates the structure that will hold all the key-value pairs.
func NewInMemoryDatabase() *InMemoryDatabase {
	return &InMemoryDatabase{
		data: make(map[string]interface{}),
	}
}

// Set stores a value in the database with a key.
// This function adds or updates a key-value pair in the database.
func (db *InMemoryDatabase) Set(key string, value interface{}) {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.data[key] = value
}

// Get retrieves a value from the database by key.
// This function returns the value associated with the given key, if it exists.
func (db *InMemoryDatabase) Get(key string) (interface{}, bool) {
	db.lock.RLock()
	defer db.lock.RUnlock()
	value, exists := db.data[key]
	return value, exists
}

// Delete removes a key-value pair from the database.
// This function deletes the entry associated with the given key.
func (db *InMemoryDatabase) Delete(key string) {
	db.lock.Lock()
	defer db.lock.Unlock()
	delete(db.data, key)
}
