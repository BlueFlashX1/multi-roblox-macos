package discord_link_parser

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	GIST_LINK = "https://gist.githubusercontent.com/Insadem/6e6c7971d1c7828fb44b182e6fd12ca0/raw"
)

// gistClient has an explicit timeout: http.Get's zero-timeout default client
// blocked the "Join Discord" caller indefinitely when the gist host hung.
var gistClient = &http.Client{Timeout: 5 * time.Second}

var cachedDiscordLink string

func parseGist() (string, error) {
	response, err := gistClient.Get(GIST_LINK)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	// Without this check a 404/500 error page was cached as the "link" and
	// served as the fallback for the rest of the session.
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gist returned status %d", response.StatusCode)
	}

	result, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	link := strings.TrimSpace(string(result))
	if link == "" {
		return "", fmt.Errorf("gist returned empty body")
	}

	return link, nil
}

func DiscordLink() string {
	result, err := parseGist()
	if err != nil {
		return cachedDiscordLink
	}

	cachedDiscordLink = result
	return result
}
