package roblox_api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// GameInfo represents Roblox game information
type GameInfo struct {
	PlaceID      int64  `json:"placeId"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	UniverseID   int64  `json:"universeId"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

// ShareLinkInfo represents resolved share link information
type ShareLinkInfo struct {
	PlaceID               int64  `json:"placeId"`
	PrivateServerLinkCode string `json:"linkCode"`
	AccessCode            string `json:"accessCode"`
	ServerID              string `json:"serverId"`
}

// secureHTTPClient creates a secure HTTP client with timeout
var secureHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	},
}

// ExtractPlaceID extracts Place ID from various Roblox URL formats
func ExtractPlaceID(urlStr string) (int64, error) {
	// Sanitize input
	urlStr = strings.TrimSpace(urlStr)

	// Handle roblox:// protocol
	if strings.HasPrefix(urlStr, "roblox://") {
		re := regexp.MustCompile(`placeId=(\d+)`)
		matches := re.FindStringSubmatch(urlStr)
		if len(matches) > 1 {
			return strconv.ParseInt(matches[1], 10, 64)
		}
		return 0, fmt.Errorf("invalid roblox:// URL format")
	}

	// Parse HTTP/HTTPS URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return 0, fmt.Errorf("invalid URL: %w", err)
	}

	// Validate domain
	if !strings.Contains(parsedURL.Host, "roblox.com") {
		return 0, fmt.Errorf("not a roblox.com URL")
	}

	// Extract from path: /games/{placeId}/...
	re := regexp.MustCompile(`/games/(\d+)`)
	matches := re.FindStringSubmatch(parsedURL.Path)
	if len(matches) > 1 {
		return strconv.ParseInt(matches[1], 10, 64)
	}

	return 0, fmt.Errorf("could not extract place ID from URL")
}

// GetGameInfo fetches game information from Roblox API
func GetGameInfo(placeID int64) (*GameInfo, error) {
	// Validate placeID
	if placeID <= 0 {
		return nil, fmt.Errorf("invalid place ID: %d", placeID)
	}

	// Step 1: Get Universe ID from Place ID
	universeID, err := getUniverseIDFromPlaceID(placeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get universe ID: %w", err)
	}

	// Step 2: Get game details
	gameURL := fmt.Sprintf("https://games.roblox.com/v1/games?universeIds=%d", universeID)

	resp, err := secureHTTPClient.Get(gameURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch game info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Data []struct {
			ID          int64  `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			RootPlaceID int64  `json:"rootPlaceId"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("game not found")
	}

	game := result.Data[0]

	// Step 3: Get thumbnail
	thumbnailURL, _ := getGameThumbnail(universeID)

	return &GameInfo{
		PlaceID:      placeID,
		Name:         game.Name,
		Description:  game.Description,
		UniverseID:   game.ID,
		ThumbnailURL: thumbnailURL,
	}, nil
}

// getUniverseIDFromPlaceID converts Place ID to Universe ID
func getUniverseIDFromPlaceID(placeID int64) (int64, error) {
	apiURL := fmt.Sprintf("https://apis.roblox.com/universes/v1/places/%d/universe", placeID)

	resp, err := secureHTTPClient.Get(apiURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result struct {
		UniverseID int64 `json:"universeId"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	return result.UniverseID, nil
}

// getGameThumbnail fetches game thumbnail URL
func getGameThumbnail(universeID int64) (string, error) {
	thumbnailURL := fmt.Sprintf("https://thumbnails.roblox.com/v1/games/icons?universeIds=%d&size=512x512&format=Png", universeID)

	resp, err := secureHTTPClient.Get(thumbnailURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Data []struct {
			ImageURL string `json:"imageUrl"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Data) > 0 && result.Data[0].ImageURL != "" {
		return result.Data[0].ImageURL, nil
	}

	return "", fmt.Errorf("no thumbnail available")
}

// DownloadThumbnail downloads and returns thumbnail image bytes
func DownloadThumbnail(thumbnailURL string) ([]byte, error) {
	if thumbnailURL == "" {
		return nil, fmt.Errorf("empty thumbnail URL")
	}

	resp, err := secureHTTPClient.Get(thumbnailURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download thumbnail: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// UserInfo represents Roblox user information
type UserInfo struct {
	UserID      int64  `json:"id"`
	Username    string `json:"name"`
	DisplayName string `json:"displayName"`
}

// UserPresence represents a user's online presence
type UserPresence struct {
	UserID           int64  `json:"userId"`
	UserPresenceType int    `json:"userPresenceType"` // 0=Offline, 1=Online, 2=InGame, 3=InStudio
	LastLocation     string `json:"lastLocation"`
	PlaceID          int64  `json:"placeId"`
	RootPlaceID      int64  `json:"rootPlaceId"`
	UniverseID       int64  `json:"universeId"`
	GameID           string `json:"gameId"`
	LastOnline       string `json:"lastOnline"`
}

// LookupUserByUsername looks up a user by their username
func LookupUserByUsername(username string) (*UserInfo, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	// Use the users API to look up by username
	apiURL := "https://users.roblox.com/v1/usernames/users"

	payload := fmt.Sprintf(`{"usernames":["%s"],"excludeBannedUsers":false}`, username)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := secureHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			RequestedUsername string `json:"requestedUsername"`
			ID                int64  `json:"id"`
			Name              string `json:"name"`
			DisplayName       string `json:"displayName"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("user not found: %s", username)
	}

	user := result.Data[0]
	return &UserInfo{
		UserID:      user.ID,
		Username:    user.Name,
		DisplayName: user.DisplayName,
	}, nil
}

// LookupUserByID looks up a user by their ID
func LookupUserByID(userID int64) (*UserInfo, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user ID")
	}

	apiURL := fmt.Sprintf("https://users.roblox.com/v1/users/%d", userID)

	resp, err := secureHTTPClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("user not found: %d", userID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var user struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
	}

	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &UserInfo{
		UserID:      user.ID,
		Username:    user.Name,
		DisplayName: user.DisplayName,
	}, nil
}

// GetUserPresence gets the online presence of one or more users
// Note: This requires authentication (cookie) to get accurate results
func GetUserPresence(userIDs []int64, cookie string) ([]UserPresence, error) {
	if len(userIDs) == 0 {
		return nil, fmt.Errorf("no user IDs provided")
	}

	apiURL := "https://presence.roblox.com/v1/presence/users"

	// Build the request body
	idsJSON, err := json.Marshal(struct {
		UserIds []int64 `json:"userIds"`
	}{UserIds: userIDs})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(idsJSON)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", ".ROBLOSECURITY="+cookie)
	}

	resp, err := secureHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get presence: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		UserPresences []UserPresence `json:"userPresences"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.UserPresences, nil
}

// GetUserAvatar gets the avatar headshot URL for a user
func GetUserAvatar(userID int64) (string, error) {
	if userID <= 0 {
		return "", fmt.Errorf("invalid user ID")
	}

	apiURL := fmt.Sprintf("https://thumbnails.roblox.com/v1/users/avatar-headshot?userIds=%d&size=150x150&format=Png", userID)

	resp, err := secureHTTPClient.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Data []struct {
			ImageURL string `json:"imageUrl"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Data) > 0 && result.Data[0].ImageURL != "" {
		return result.Data[0].ImageURL, nil
	}

	return "", fmt.Errorf("no avatar available")
}

// ResolveShareLink resolves a share link code to get the actual private server details
// This is needed because share codes (from roblox.com/share?code=XXX) are different
// from the direct privateServerLinkCode used in game URLs
func ResolveShareLink(shareCode string, cookie string) (*ShareLinkInfo, error) {
	if shareCode == "" {
		return nil, fmt.Errorf("empty share code")
	}

	// Try the share-links API
	apiURL := "https://apis.roblox.com/share-links/v1/resolve-link"

	payload := fmt.Sprintf(`{"linkId":"%s","linkType":"Server"}`, shareCode)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", ".ROBLOSECURITY="+cookie)
	}

	resp, err := secureHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve share link: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Log the response for debugging
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("share API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Try to parse the response - the structure might vary
	var result struct {
		PlaceID               int64  `json:"placeId"`
		PrivateServerLinkCode string `json:"privateServerLinkCode"`
		LinkCode              string `json:"linkCode"`
		AccessCode            string `json:"accessCode"`
		PrivateServerID       int64  `json:"privateServerId"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		// Try alternative parsing
		var altResult map[string]interface{}
		if err2 := json.Unmarshal(body, &altResult); err2 == nil {
			// Log what we got for debugging
			return nil, fmt.Errorf("share link response format unknown: %s", string(body))
		}
		return nil, fmt.Errorf("failed to parse share link response: %w", err)
	}

	info := &ShareLinkInfo{
		PlaceID: result.PlaceID,
	}

	// Use whichever link code field is populated
	if result.PrivateServerLinkCode != "" {
		info.PrivateServerLinkCode = result.PrivateServerLinkCode
	} else if result.LinkCode != "" {
		info.PrivateServerLinkCode = result.LinkCode
	} else if result.AccessCode != "" {
		info.AccessCode = result.AccessCode
	}

	if result.PrivateServerID > 0 {
		info.ServerID = fmt.Sprintf("%d", result.PrivateServerID)
	}

	return info, nil
}

// GetPrivateServerJoinScript gets the join script for a private server
// This is an alternative approach that mimics what the browser does
func GetPrivateServerJoinScript(placeID int64, accessCode string, cookie string) (string, error) {
	// Try using the games API to get join info
	apiURL := fmt.Sprintf("https://gamejoin.roblox.com/v1/join-private-game")

	payload := fmt.Sprintf(`{"placeId":%d,"accessCode":"%s"}`, placeID, accessCode)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", ".ROBLOSECURITY="+cookie)
	}

	resp, err := secureHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get join script: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("join API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		JobID      string `json:"jobId"`
		Status     int    `json:"status"`
		JoinScript string `json:"joinScript"`
		AuthTicket string `json:"authenticationTicket"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse join response: %w", err)
	}

	if result.JoinScript != "" {
		return result.JoinScript, nil
	}
	if result.JobID != "" {
		return result.JobID, nil
	}

	return "", fmt.Errorf("no join script in response: %s", string(body))
}
