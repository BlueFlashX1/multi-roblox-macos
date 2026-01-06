package friends_manager

import (
	"encoding/json"
	"fmt"
	"insadem/multi_roblox_macos/internal/logger"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PresenceType represents the user's current status
type PresenceType int

const (
	PresenceOffline PresenceType = iota
	PresenceOnline
	PresenceInGame
	PresenceInStudio
)

// Friend represents a saved friend
type Friend struct {
	UserID      int64     `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name,omitempty"`
	AddedAt     time.Time `json:"added_at"`
	Notes       string    `json:"notes,omitempty"`
}

// FriendStatus represents current status of a friend
type FriendStatus struct {
	UserID      int64        `json:"user_id"`
	Presence    PresenceType `json:"presence"`
	LastOnline  time.Time    `json:"last_online,omitempty"`
	PlaceID     int64        `json:"place_id,omitempty"`
	GameName    string       `json:"game_name,omitempty"`
	LastUpdated time.Time    `json:"last_updated"`
}

// FriendsConfig stores all saved friends
type FriendsConfig struct {
	Friends []Friend `json:"friends"`
}

var (
	statusCache     = make(map[int64]FriendStatus)
	statusCacheLock sync.RWMutex
)

// GetConfigPath returns the path to the friends config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(homeDir, "Library", "Application Support", "multi_roblox_macos")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(configDir, "friends.json"), nil
}

// LoadFriends loads friends from config file
func LoadFriends() ([]Friend, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Friend{}, nil
		}
		return nil, err
	}

	var config FriendsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config.Friends, nil
}

// SaveFriends saves friends to config file
func SaveFriends(friends []Friend) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	config := FriendsConfig{Friends: friends}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600) // Secure permissions - owner only
}

// AddFriend adds a new friend
func AddFriend(userID int64, username, displayName string) error {
	friends, err := LoadFriends()
	if err != nil {
		return err
	}

	// Check if already exists
	for _, f := range friends {
		if f.UserID == userID {
			return fmt.Errorf("friend with user ID %d already exists", userID)
		}
	}

	friend := Friend{
		UserID:      userID,
		Username:    username,
		DisplayName: displayName,
		AddedAt:     time.Now(),
	}

	friends = append(friends, friend)
	logger.LogInfo("Added friend: %s (ID: %d)", username, userID)
	return SaveFriends(friends)
}

// RemoveFriend removes a friend by user ID
func RemoveFriend(userID int64) error {
	friends, err := LoadFriends()
	if err != nil {
		return err
	}

	var newFriends []Friend
	found := false
	for _, f := range friends {
		if f.UserID == userID {
			found = true
			continue
		}
		newFriends = append(newFriends, f)
	}

	if !found {
		return fmt.Errorf("friend not found")
	}

	logger.LogInfo("Removed friend with ID: %d", userID)
	return SaveFriends(newFriends)
}

// UpdateFriendNotes updates notes for a friend
func UpdateFriendNotes(userID int64, notes string) error {
	friends, err := LoadFriends()
	if err != nil {
		return err
	}

	for i, f := range friends {
		if f.UserID == userID {
			friends[i].Notes = notes
			return SaveFriends(friends)
		}
	}

	return fmt.Errorf("friend not found")
}

// GetCachedStatus returns cached status for a friend
func GetCachedStatus(userID int64) (FriendStatus, bool) {
	statusCacheLock.RLock()
	defer statusCacheLock.RUnlock()
	status, ok := statusCache[userID]
	return status, ok
}

// UpdateCachedStatus updates the cached status for a friend
func UpdateCachedStatus(status FriendStatus) {
	statusCacheLock.Lock()
	defer statusCacheLock.Unlock()
	status.LastUpdated = time.Now()
	statusCache[status.UserID] = status
}

// GetAllCachedStatuses returns all cached statuses
func GetAllCachedStatuses() map[int64]FriendStatus {
	statusCacheLock.RLock()
	defer statusCacheLock.RUnlock()

	result := make(map[int64]FriendStatus)
	for k, v := range statusCache {
		result[k] = v
	}
	return result
}

// PresenceString returns a human-readable presence string
func (p PresenceType) String() string {
	switch p {
	case PresenceOffline:
		return "Offline"
	case PresenceOnline:
		return "Online"
	case PresenceInGame:
		return "In Game"
	case PresenceInStudio:
		return "In Studio"
	default:
		return "Unknown"
	}
}

// PresenceIcon returns an emoji for the presence
func (p PresenceType) Icon() string {
	switch p {
	case PresenceOffline:
		return "⚫"
	case PresenceOnline:
		return "🟢"
	case PresenceInGame:
		return "🎮"
	case PresenceInStudio:
		return "🔧"
	default:
		return "❓"
	}
}
