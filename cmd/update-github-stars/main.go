package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"awesome-ai-agents/internal/projectdata"
)

var repoPattern = regexp.MustCompile(`(?i)^https?://(?:www\.)?github\.com/([^/]+)/([^/#?]+)`)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}
func run() error {
	root := projectdata.Root()
	dataPath := root + "/awesome-agents.json"
	histPath := root + "/github-stars-history.json"
	var data projectdata.Data
	if e := projectdata.ReadJSON(dataPath, &data); e != nil {
		return e
	}
	history := projectdata.History{Snapshots: []projectdata.HistoryRow{}}
	if e := projectdata.ReadJSON(histPath, &history); e != nil && !os.IsNotExist(e) {
		return e
	}
	today := time.Now().UTC().Format("2006-01-02")
	repos := map[string][]*projectdata.Source{}
	for i := range data.Agents {
		for j := range data.Agents[i].Sources {
			s := &data.Agents[i].Sources[j]
			if s.Source != "github" {
				continue
			}
			m := repoPattern.FindStringSubmatch(s.SourceURL)
			if len(m) < 3 {
				continue
			}
			repo := strings.ToLower(m[1] + "/" + strings.TrimSuffix(m[2], ".git"))
			repos[repo] = append(repos[repo], s)
		}
	}
	keys := make([]string, 0, len(repos))
	for k := range repos {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	client := &http.Client{Timeout: 30 * time.Second}
	failures := 0
	for _, key := range keys {
		req, e := http.NewRequest(http.MethodGet, "https://api.github.com/repos/"+key, nil)
		if e != nil {
			return e
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if t := projectdata.GitHubToken(root); t != "" {
			req.Header.Set("Authorization", "Bearer "+t)
		}
		resp, e := client.Do(req)
		if e != nil {
			failures++
			setFailure(repos[key], "check_failed", fmt.Sprintf("GitHub API request failed: %v; retry next run.", e))
			for _, s := range repos[key] {
				s.RepositoryCheckedAt = time.Now().UTC().Format("2006-01-02T15:04:05Z")
			}
			fmt.Fprintf(os.Stderr, "WARN %s: %v\n", key, e)
			continue
		}
		checked := time.Now().UTC().Format("2006-01-02T15:04:05Z")
		if resp.StatusCode == 200 {
			var gh struct {
				Stars    int  `json:"stargazers_count"`
				Archived bool `json:"archived"`
			}
			e = json.NewDecoder(resp.Body).Decode(&gh)
			resp.Body.Close()
			if e != nil {
				return e
			}
			status := "active"
			if gh.Archived {
				status = "archived"
			}
			for _, s := range repos[key] {
				n := gh.Stars
				s.Stars = &n
				s.StarsLastUpdated = &checked
				s.RepositoryCheckedAt = checked
				s.RepositoryStatus = status
				s.RepositoryStatusDetail = ""
			}
			n := gh.Stars
			upsertSnapshot(&history, projectdata.HistoryRow{Repository: key, Date: today, Stars: &n, Status: status})
		} else if resp.StatusCode == 404 {
			resp.Body.Close()
			setFailure(repos[key], "not_found", "GitHub returned 404; repository may be deleted, private, or renamed.")
			for _, s := range repos[key] {
				s.RepositoryCheckedAt = checked
			}
			upsertSnapshot(&history, projectdata.HistoryRow{Repository: key, Date: today, Status: "not_found"})
		} else {
			code := resp.StatusCode
			resp.Body.Close()
			failures++
			setFailure(repos[key], "check_failed", fmt.Sprintf("GitHub API returned HTTP %d; retry next run.", code))
			for _, s := range repos[key] {
				s.RepositoryCheckedAt = checked
			}
			fmt.Fprintf(os.Stderr, "WARN %s: HTTP %d\n", key, code)
		}
	}
	sort.Slice(history.Snapshots, func(i, j int) bool {
		a, b := history.Snapshots[i], history.Snapshots[j]
		if strings.ToLower(a.Repository) != strings.ToLower(b.Repository) {
			return strings.ToLower(a.Repository) < strings.ToLower(b.Repository)
		}
		return a.Date < b.Date
	})
	if e := projectdata.WriteJSON(histPath, history, true); e != nil {
		return e
	}
	if e := projectdata.WriteJSON(dataPath, data, true); e != nil {
		return e
	}
	fmt.Printf("Snapshot date: %s; repositories checked: %d; API failures: %d\n", today, len(repos), failures)
	if failures > 0 {
		return fmt.Errorf("%d repository checks failed", failures)
	}
	return nil
}
func setFailure(sources []*projectdata.Source, status, detail string) {
	for _, s := range sources {
		s.RepositoryStatus = status
		s.RepositoryStatusDetail = detail
	}
}

func upsertSnapshot(history *projectdata.History, row projectdata.HistoryRow) {
	for i := range history.Snapshots {
		if strings.EqualFold(history.Snapshots[i].Repository, row.Repository) && history.Snapshots[i].Date == row.Date {
			history.Snapshots[i] = row
			return
		}
	}
	history.Snapshots = append(history.Snapshots, row)
}
