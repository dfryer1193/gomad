package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

type FileMetadata struct {
	RepoName string
	Path     string
	Commit   string
}

type minimalGitHubFileData struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	Size     int    `json:"size"`
}

type GitFileFetcher struct {
	client *http.Client
}

var (
	fileFetcher *GitFileFetcher
	once        sync.Once
)

func GetGitFileFetcher() *GitFileFetcher {
	once.Do(func() {
		fileFetcher = &GitFileFetcher{
			client: &http.Client{
				Timeout: 10 * time.Second,
				Transport: &http.Transport{
					MaxIdleConns:        10,
					IdleConnTimeout:     30 * time.Second,
					DisableCompression:  true,
					MaxIdleConnsPerHost: 10,
					DisableKeepAlives:   true,
					ForceAttemptHTTP2:   true,
				},
			},
		}
	})

	return fileFetcher
}

func (f *GitFileFetcher) FetchRawGitFile(metadata FileMetadata) (string, error) {
	fetchUrl := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s?ref=%s", metadata.RepoName, metadata.Path, metadata.Commit)
	req, err := http.NewRequest("GET", fetchUrl, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create file fetch request: %w", err)
	}

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("token %s", token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch file: %s %w", fetchUrl, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("file too large to fetch: %s", fetchUrl)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d for file: %s", resp.StatusCode, fetchUrl)
	}

	var fileContent minimalGitHubFileData
	if err := json.NewDecoder(resp.Body).Decode(&fileContent); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if fileContent.Encoding != "base64" {
		return "", fmt.Errorf("unsupported file encoding: %s", fileContent.Encoding)
	}

	decoded, err := base64.StdEncoding.DecodeString(fileContent.Content)
	if err != nil {
		return "", fmt.Errorf("failed to decode file content: %w", err)
	}

	return string(decoded), nil
}
