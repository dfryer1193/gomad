package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type GitFileFetcher struct {
	token   string
	baseURL string
}

type FileContent struct {
	Content  string
	Path     string
	Filename string
}

type githubAPIResponse struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

func NewGitFileFetcher() *GitFileFetcher {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		panic("GITHUB_TOKEN environment variable is required")
	}

	return &GitFileFetcher{
		token:   token,
		baseURL: "https://api.github.com",
	}
}

// FetchFile retrieves a file's content from a GitHub repository
func (s *GitFileFetcher) FetchFile(repo string, filepath string) (*FileContent, error) {
	url := fmt.Sprintf("%s/repos/%s/contents/%s", s.baseURL, repo, filepath)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var apiResp githubAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(apiResp.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to decode content: %w", err)
	}

	return &FileContent{
		Content:  string(decoded),
		Path:     filepath,
		Filename: filepath, // You might want to extract just the filename
	}, nil
}

// FetchFiles retrieves multiple files from a repository
func (s *GitFileFetcher) FetchFiles(repo string, filepaths []string) ([]*FileContent, error) {
	var files []*FileContent
	var errors []error

	for _, path := range filepaths {
		file, err := s.FetchFile(repo, path)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to fetch %s: %w", path, err))
			continue
		}
		files = append(files, file)
	}

	if len(errors) > 0 {
		// Combine errors into a single error
		errStr := "failed to fetch some files:"
		for _, err := range errors {
			errStr += "\n" + err.Error()
		}
		return files, fmt.Errorf(errStr)
	}

	return files, nil
}
