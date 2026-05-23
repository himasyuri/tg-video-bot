package cache

import (
	"strconv"
	"time"

	"github.com/patrickmn/go-cache"
)

// UserStats stores the quantity of downloads and conversions for a user.
type UserStats struct {
	Downloads     int
	Conversions   int
	RequestCount  int
	LastResetTime time.Time
}

// Cache is a wrapper around go-cache to store user statistics.
type Cache struct {
	store *cache.Cache
}

// New creates a new in-memory cache with the specified expiration and cleanup intervals.
func New(defaultExpiration, cleanupInterval time.Duration) *Cache {
	return &Cache{
		store: cache.New(defaultExpiration, cleanupInterval),
	}
}

// GetUserStats retrieves the statistics for a specific user.
func (c *Cache) GetUserStats(userID int64) (UserStats, bool) {
	key := strconv.FormatInt(userID, 10)
	val, found := c.store.Get(key)
	if !found {
		return UserStats{LastResetTime: time.Now()}, false
	}
	return val.(UserStats), true
}

// SetUserStats stores the statistics for a specific user.
func (c *Cache) SetUserStats(userID int64, stats UserStats) {
	key := strconv.FormatInt(userID, 10)
	c.store.Set(key, stats, cache.DefaultExpiration)
}

// CheckRateLimit checks if the user has exceeded 5 requests in 3 hours.
// It returns true if the request is allowed, false otherwise.
func (c *Cache) CheckRateLimit(userID int64) bool {
	stats, _ := c.GetUserStats(userID)
	now := time.Now()

	// Reset counter if more than 3 hours passed since LastResetTime
	if now.Sub(stats.LastResetTime) > 3*time.Hour {
		stats.RequestCount = 0
		stats.LastResetTime = now
	}

	if stats.RequestCount >= 5 {
		return false
	}

	stats.RequestCount++
	c.SetUserStats(userID, stats)
	return true
}

// IncrementDownloads increases the download count for a user.
func (c *Cache) IncrementDownloads(userID int64) {
	stats, _ := c.GetUserStats(userID)
	stats.Downloads++
	c.SetUserStats(userID, stats)
}

// IncrementConversions increases the conversion count for a user.
func (c *Cache) IncrementConversions(userID int64) {
	stats, _ := c.GetUserStats(userID)
	stats.Conversions++
	c.SetUserStats(userID, stats)
}
