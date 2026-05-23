package cache

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	c := New(5*time.Minute, 10*time.Minute)
	userID := int64(12345)

	// Test IncrementDownloads
	c.IncrementDownloads(userID)
	stats, found := c.GetUserStats(userID)
	if !found {
		t.Errorf("expected stats to be found")
	}
	if stats.Downloads != 1 {
		t.Errorf("expected 1 download, got %d", stats.Downloads)
	}

	// Test Rate Limiting
	// Reset to known state
	c.store.Delete("12345")
	
	// First 5 requests should be allowed
	for i := 1; i <= 5; i++ {
		if !c.CheckRateLimit(userID) {
			t.Errorf("request %d should have been allowed", i)
		}
	}

	// 6th request should be blocked
	if c.CheckRateLimit(userID) {
		t.Error("6th request should have been blocked")
	}

	// Test time reset (manually adjust stats in store for testing reset)
	stats, _ = c.GetUserStats(userID)
	stats.LastResetTime = time.Now().Add(-4 * time.Hour) // Set last reset to 4 hours ago
	c.SetUserStats(userID, stats)

	if !c.CheckRateLimit(userID) {
		t.Error("request should be allowed after 3 hour reset period")
	}
}
