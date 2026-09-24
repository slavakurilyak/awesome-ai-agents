// Command triage fetches incremental open GitHub issues and pull requests for
// the Codex classification workflow. It never writes to GitHub.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"awesome-ai-agents/internal/projectdata"
)

var repoID = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

type candidate struct {
	Kind      string   `json:"kind"`
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	URL       string   `json:"url"`
	UpdatedAt string   `json:"updated_at"`
	Author    string   `json:"author"`
	Labels    []string `json:"labels"`
	Body      string   `json:"body"`
}

type triageState struct {
	LastProcessedAt string `json:"last_processed_at"`
}

func main() {
	var err error
	root := projectdata.Root()
	if len(os.Args) < 2 {
		err = errors.New("usage: go run ./cmd/triage {fetch|ack RFC3339-cursor}")
	} else {
		switch os.Args[1] {
		case "fetch":
			err = fetch(root)
		case "ack":
			if len(os.Args) != 3 {
				err = errors.New("ack requires the next_cursor returned by fetch")
			} else {
				err = acknowledge(root, os.Args[2])
			}
		default:
			err = fmt.Errorf("unknown command %q; expected fetch or ack", os.Args[1])
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}

func fetch(root string) error {
	repository := os.Getenv("GITHUB_REPOSITORY")
	if repository == "" {
		repository = "slavakurilyak/awesome-ai-agents"
	}
	if !repoID.MatchString(repository) {
		return errors.New("GITHUB_REPOSITORY must be owner/repo")
	}
	statePath := filepath.Join(root, ".curation-triage-state.json")
	state := triageState{}
	if bytes, err := os.ReadFile(statePath); err == nil {
		if err := json.Unmarshal(bytes, &state); err != nil {
			return fmt.Errorf("decode triage state: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	nextCursor := state.LastProcessedAt
	client := &http.Client{Timeout: 30 * time.Second}
	items := make([]candidate, 0)
	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("https://api.github.com/repos/%s/issues?state=open&per_page=100&page=%d", repository, page)
		request, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return err
		}
		request.Header.Set("Accept", "application/vnd.github+json")
		request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if token := projectdata.GitHubToken(root); token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		response, err := client.Do(request)
		if err != nil {
			return err
		}
		if response.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
			response.Body.Close()
			return fmt.Errorf("GitHub issues API HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
		}
		var pageItems []struct {
			Number    int    `json:"number"`
			Title     string `json:"title"`
			HTMLURL   string `json:"html_url"`
			UpdatedAt string `json:"updated_at"`
			Body      string `json:"body"`
			Pull      *any   `json:"pull_request"`
			User      struct {
				Login string `json:"login"`
			} `json:"user"`
			Labels []struct {
				Name string `json:"name"`
			} `json:"labels"`
		}
		err = json.NewDecoder(response.Body).Decode(&pageItems)
		response.Body.Close()
		if err != nil {
			return err
		}
		for _, item := range pageItems {
			if state.LastProcessedAt != "" && item.UpdatedAt <= state.LastProcessedAt {
				continue
			}
			if item.UpdatedAt > nextCursor {
				nextCursor = item.UpdatedAt
			}
			kind := "issue"
			if item.Pull != nil {
				kind = "pull_request"
			}
			labels := make([]string, 0, len(item.Labels))
			for _, label := range item.Labels {
				labels = append(labels, label.Name)
			}
			items = append(items, candidate{kind, item.Number, item.Title, item.HTMLURL, item.UpdatedAt, item.User.Login, labels, item.Body})
		}
		if len(pageItems) < 100 {
			break
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt == items[j].UpdatedAt {
			return items[i].Number < items[j].Number
		}
		return items[i].UpdatedAt < items[j].UpdatedAt
	})
	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"repository": repository, "previous_cursor": state.LastProcessedAt,
		"next_cursor": nextCursor, "open_items": items,
	})
}

func acknowledge(root, cursor string) error {
	if _, err := time.Parse(time.RFC3339, cursor); err != nil {
		return fmt.Errorf("invalid triage cursor: %w", err)
	}
	return writeJSON(filepath.Join(root, ".curation-triage-state.json"), triageState{LastProcessedAt: cursor})
}

func writeJSON(path string, value any) error {
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(bytes, '\n'), 0644)
}
