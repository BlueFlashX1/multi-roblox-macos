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
	PlaceID     int64  `json:"placeId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	UniverseID  int64  `json:"universeId"`
	ThumbnailURL string `json:"thumbnailUrl"`
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
