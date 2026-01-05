package thumbnail_cache

import (
	"crypto/md5"
	"fmt"
	"insadem/multi_roblox_macos/internal/roblox_api"
	"os"
	"path/filepath"
)

// GetCachePath returns the thumbnail cache directory
func GetCachePath() string {
	home, _ := os.UserHomeDir()
	cacheDir := filepath.Join(home, "Library", "Caches", "multi_roblox_macos", "thumbnails")
	os.MkdirAll(cacheDir, 0755)
	return cacheDir
}

// GetThumbnailPath returns the local path for a cached thumbnail
func GetThumbnailPath(thumbnailURL string) string {
	// Use MD5 hash of URL as filename
	hash := md5.Sum([]byte(thumbnailURL))
	filename := fmt.Sprintf("%x.png", hash)
	return filepath.Join(GetCachePath(), filename)
}

// DownloadAndCacheThumbnail downloads a thumbnail and caches it locally
func DownloadAndCacheThumbnail(thumbnailURL string) (string, error) {
	if thumbnailURL == "" {
		return "", fmt.Errorf("empty thumbnail URL")
	}

	cachePath := GetThumbnailPath(thumbnailURL)

	// Check if already cached
	if _, err := os.Stat(cachePath); err == nil {
		return cachePath, nil
	}

	// Download thumbnail
	data, err := roblox_api.DownloadThumbnail(thumbnailURL)
	if err != nil {
		return "", err
	}

	// Save to cache
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return "", err
	}

	return cachePath, nil
}

// GetCachedThumbnail returns the cached thumbnail path if it exists
func GetCachedThumbnail(thumbnailURL string) (string, bool) {
	if thumbnailURL == "" {
		return "", false
	}

	cachePath := GetThumbnailPath(thumbnailURL)
	if _, err := os.Stat(cachePath); err == nil {
		return cachePath, true
	}

	return "", false
}

// ClearCache removes all cached thumbnails
func ClearCache() error {
	cacheDir := GetCachePath()
	return os.RemoveAll(cacheDir)
}
