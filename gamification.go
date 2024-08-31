package main

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Badge represents an achievement earned by the user.
type Badge struct {
	Name        string
	Description string
	Criteria    func(user *User) bool // Criteria function to dynamically assign badges
}

// User represents a participant in the blockchain network.
type User struct {
	Address    string
	Points     int
	Level      int
	Badges     map[string]Badge
	LastActive time.Time
	RewardLog  map[string]time.Time // Logs when the user last received rewards for specific activities
}

// Gamification handles user rewards, levels, and badges.
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
type Leaderboard struct {
	Users []*User
	lock  sync.Mutex
}

// NewGamification initializes a new Gamification system with a database.
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

func (g *Gamification) loadOrCreateUser(address string) (*User, error) {
	if user, exists := g.Users[address]; exists {
		return user, nil
	}

	if value, exists := g.db.Get(address); exists {
		user := value.(*User)
		g.Users[address] = user
		return user, nil
	}

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

func (g *Gamification) checkLevelUp(user *User) {
	for i, threshold := range g.Levels {
		if user.Points >= threshold && user.Level <= i+1 {
			user.Level = i + 2 // Levels are 1-based, thresholds are 0-based
			fmt.Printf("User %s leveled up to level %d!\n", user.Address, user.Level)
		}
	}
}

func (g *Gamification) assignBadges(user *User) {
	for _, badge := range g.Badges {
		if _, hasBadge := user.Badges[badge.Name]; !hasBadge && badge.Criteria(user) {
			user.Badges[badge.Name] = badge
			fmt.Printf("User %s earned badge: %s\n", user.Address, badge.Name)
		}
	}
}

// RewardFullNode rewards users based on full node uptime.
func (g *Gamification) RewardFullNode(address string, uptime time.Duration) error {
	points := int(uptime.Hours()) * 10 // Example: 10 points per hour of uptime
	return g.RewardUser(address, points, "full-node")
}

// RewardLiquidityProvider rewards users for providing liquidity.
func (g *Gamification) RewardLiquidityProvider(address string, liquidityProvided int) error {
	points := liquidityProvided / 100 // Example: 1 point per 100 units of liquidity
	return g.RewardUser(address, points, "liquidity")
}

// RewardTransactionFees rewards users for contributing transaction fees.
func (g *Gamification) RewardTransactionFees(address string, feesContributed int) error {
	points := feesContributed / 10 // Example: 1 point per 10 units of fees
	return g.RewardUser(address, points, "transaction-fees")
}

func (g *Gamification) DetectSuspiciousPatterns(user *User) error {
	activityFrequency := time.Since(user.LastActive)
	if activityFrequency < 1*time.Minute {
		return errors.New("suspicious activity detected: too frequent actions")
	}
	return nil
}

func (g *Gamification) EnforceCooldown(user *User, activity string) error {
	if cooldown, ok := g.Cooldowns[activity]; ok {
		if time.Since(user.RewardLog[activity]) < cooldown {
			return fmt.Errorf("cooldown period for %s not met", activity)
		}
	}
	return nil
}

// Leaderboard management
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
type InMemoryDatabase struct {
	data map[string]interface{}
	lock sync.RWMutex
}

// NewInMemoryDatabase initializes a new in-memory database.
func NewInMemoryDatabase() *InMemoryDatabase {
	return &InMemoryDatabase{
		data: make(map[string]interface{}),
	}
}

// Set stores a value in the database with a key.
func (db *InMemoryDatabase) Set(key string, value interface{}) {
	db.lock.Lock()
	defer db.lock.Unlock()
	db.data[key] = value
}

// Get retrieves a value from the database by key.
func (db *InMemoryDatabase) Get(key string) (interface{}, bool) {
	db.lock.RLock()
	defer db.lock.RUnlock()
	value, exists := db.data[key]
	return value, exists
}

// Delete removes a key-value pair from the database.
func (db *InMemoryDatabase) Delete(key string) {
	db.lock.Lock()
	defer db.lock.Unlock()
	delete(db.data, key)
}
