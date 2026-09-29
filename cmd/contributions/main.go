// Command contributions assigns stable project IDs, validates the public
// contribution ledger, and prepares read-only historical review candidates.
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
	"strconv"
	"strings"
	"sync"
	"time"

	"awesome-ai-agents/internal/contributions"
	"awesome-ai-agents/internal/projectdata"
)

const defaultRepository = "slavakurilyak/awesome-ai-agents"

type actor struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

type issue struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	State       string `json:"state"`
	CreatedAt   string `json:"created_at"`
	ClosedAt    string `json:"closed_at"`
	HTMLURL     string `json:"html_url"`
	User        actor  `json:"user"`
	PullRequest *struct {
		URL string `json:"url"`
	} `json:"pull_request"`
}

type pull struct {
	Number    int     `json:"number"`
	Title     string  `json:"title"`
	Body      string  `json:"body"`
	CreatedAt string  `json:"created_at"`
	ClosedAt  string  `json:"closed_at"`
	HTMLURL   string  `json:"html_url"`
	MergedAt  *string `json:"merged_at"`
	User      actor   `json:"user"`
}

type pullFile struct {
	Filename string `json:"filename"`
	Patch    string `json:"patch"`
}

type candidate struct {
	ProjectID           string   `json:"project_id"`
	Project             string   `json:"project"`
	IssueNumber         int      `json:"issue_number"`
	IssueURL            string   `json:"issue_url"`
	AuthorID            int64    `json:"author_id"`
	Author              string   `json:"author"`
	SubmittedAt         string   `json:"submitted_at"`
	FounderTeamClaim    bool     `json:"founder_team_claim_self_reported"`
	MaintainerClaim     bool     `json:"maintainer_claim_self_reported"`
	Accepted            bool     `json:"accepted"`
	MergedPulls         []string `json:"merged_pull_requests,omitempty"`
	RelatedCatalogPulls []string `json:"related_catalog_pull_requests,omitempty"`
}

type catalogPullCandidate struct {
	Number          int      `json:"number"`
	Title           string   `json:"title"`
	URL             string   `json:"url"`
	AuthorID        int64    `json:"author_id"`
	Author          string   `json:"author"`
	MergedAt        string   `json:"merged_at,omitempty"`
	ClosedAt        string   `json:"closed_at"`
	Files           []string `json:"catalog_files"`
	AddedProjects   []string `json:"added_current_projects,omitempty"`
	MatchedProjects []string `json:"matched_current_projects,omitempty"`
}

type report struct {
	Repository          string                 `json:"repository"`
	GeneratedAt         string                 `json:"generated_at"`
	ClosedIssuesScanned int                    `json:"closed_issues_scanned"`
	ClosedPullsScanned  int                    `json:"closed_pull_requests_scanned"`
	PullRequestsScanned int                    `json:"pull_requests_scanned"`
	Candidates          []candidate            `json:"submission_candidates"`
	CatalogPulls        []catalogPullCandidate `json:"catalog_pull_requests"`
}

var repositoryURLs = regexp.MustCompile(`(?i)https?://(?:www\.)?(?:github\.com|gitlab\.com|codeberg\.org)/[^\s<>"']+`)
var founderClaim = regexp.MustCompile(`(?im)^\s*[-*]\s*\[x\]\s*I am (?:a )?(?:founder|team member)\b`)
var maintainerClaim = regexp.MustCompile(`(?i)\bI (?:currently )?maintain(?:s|ing)?\b|\bI(?:'m| am) (?:a )?maintainer\b`)
var issueReference = regexp.MustCompile(`(?i)(?:close(?:s|d)?(?: issue)?|fix(?:es|ed)?(?: issue)?|resolve(?:s|d)?(?: issue)?|from issue|issue)\s*#?(\d+)`)
var issueURLReference = regexp.MustCompile(`(?i)github\.com/[^/]+/[^/]+/issues/(\d+)`)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	root := projectdata.Root()
	if len(args) == 0 {
		return errors.New("usage: go run ./cmd/contributions {assign-ids|validate|backfill|import-reviewed}")
	}
	switch args[0] {
	case "assign-ids":
		if len(args) != 1 {
			return errors.New("assign-ids takes no additional arguments")
		}
		return assignIDs(root)
	case "validate":
		switch {
		case len(args) == 1:
			return validate(root, "")
		case len(args) == 3 && args[1] == "--project" && args[2] != "":
			return validate(root, args[2])
		default:
			return errors.New("usage: validate [--project \"Project name\"]")
		}
	case "backfill":
		if len(args) != 1 {
			return errors.New("backfill takes no additional arguments")
		}
		return backfill(root)
	case "import-reviewed":
		return importReviewed(root, args[1:])
	default:
		return fmt.Errorf("unknown command %q; expected assign-ids, validate, backfill, or import-reviewed", args[0])
	}
}

func importReviewed(root string, args []string) error {
	if len(args) < 2 {
		return errors.New("usage: go run ./cmd/contributions import-reviewed <backfill-report.json> <reviewed-issue-number> [issue-number ...]")
	}
	selected := make(map[int]bool, len(args)-1)
	for _, raw := range args[1:] {
		number, err := strconv.Atoi(raw)
		if err != nil || number <= 0 {
			return fmt.Errorf("invalid issue number %q", raw)
		}
		selected[number] = true
	}
	bytes, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	var history report
	if err := json.Unmarshal(bytes, &history); err != nil {
		return fmt.Errorf("decode backfill report: %w", err)
	}
	ledgerPath := filepath.Join(root, "contributions.json")
	ledger, err := contributions.Read(ledgerPath)
	if err != nil {
		return err
	}
	projects := make([]contributions.Project, 0, len(ledger.Projects)+len(history.Candidates))
	projects = append(projects, ledger.Projects...)
	ledger.Projects = projects
	byProject := make(map[string]*contributions.Project, len(ledger.Projects))
	for i := range ledger.Projects {
		byProject[ledger.Projects[i].ProjectID] = &ledger.Projects[i]
	}
	seenIssues := map[int]bool{}
	imported := 0
	for _, candidate := range history.Candidates {
		if !selected[candidate.IssueNumber] {
			continue
		}
		seenIssues[candidate.IssueNumber] = true
		if !candidate.Accepted || len(candidate.MergedPulls) == 0 {
			return fmt.Errorf("issue #%d is not an accepted backfill candidate", candidate.IssueNumber)
		}
		project := byProject[candidate.ProjectID]
		if project == nil {
			ledger.Projects = append(ledger.Projects, contributions.Project{ProjectID: candidate.ProjectID})
			project = &ledger.Projects[len(ledger.Projects)-1]
			byProject[candidate.ProjectID] = project
		}
		duplicate := false
		for _, existing := range project.SubmittedBy {
			if existing.EvidenceURL == candidate.IssueURL {
				duplicate = true
			}
		}
		if !duplicate {
			project.SubmittedBy = append(project.SubmittedBy, contributions.Credit{
				GitHubID: candidate.AuthorID, Login: candidate.Author, EvidenceURL: candidate.IssueURL,
				AcceptanceURL: candidate.MergedPulls[0], At: candidate.SubmittedAt, Status: "accepted_submission",
				FounderTeamClaim: candidate.FounderTeamClaim,
			})
			imported++
		}
		if candidate.MaintainerClaim {
			alreadyMaintainer := false
			for _, existing := range project.MaintainedBy {
				if existing.GitHubID == candidate.AuthorID && existing.EvidenceURL == candidate.IssueURL {
					alreadyMaintainer = true
				}
			}
			if !alreadyMaintainer {
				project.MaintainedBy = append(project.MaintainedBy, contributions.Credit{
					GitHubID: candidate.AuthorID, Login: candidate.Author, EvidenceURL: candidate.IssueURL,
					At: candidate.SubmittedAt, Status: "self_reported",
				})
			}
		}
	}
	for number := range selected {
		if !seenIssues[number] {
			return fmt.Errorf("issue #%d does not match a project in the backfill report", number)
		}
	}
	sort.Slice(ledger.Projects, func(i, j int) bool { return ledger.Projects[i].ProjectID < ledger.Projects[j].ProjectID })
	for i := range ledger.Projects {
		sort.Slice(ledger.Projects[i].SubmittedBy, func(a, b int) bool {
			return ledger.Projects[i].SubmittedBy[a].At < ledger.Projects[i].SubmittedBy[b].At
		})
		sort.Slice(ledger.Projects[i].MaintainedBy, func(a, b int) bool {
			return ledger.Projects[i].MaintainedBy[a].At < ledger.Projects[i].MaintainedBy[b].At
		})
	}
	catalog, err := projectdata.LoadData(filepath.Join(root, "awesome-agents.json"))
	if err != nil {
		return err
	}
	if err := contributions.Validate(catalog, ledger); err != nil {
		return err
	}
	if err := contributions.Write(ledgerPath, ledger); err != nil {
		return err
	}
	fmt.Printf("Imported %d reviewed submission credits from %d selected issue(s).\n", imported, len(selected))
	return nil
}

func assignIDs(root string) error {
	path := filepath.Join(root, "awesome-agents.json")
	var data projectdata.Data
	if err := projectdata.ReadJSON(path, &data); err != nil {
		return err
	}
	assigned := 0
	for i := range data.Agents {
		if data.Agents[i].ID != "" {
			continue
		}
		if projectdata.EnsureProjectID(&data.Agents[i]) {
			assigned++
		}
	}
	if err := projectdata.Validate(data); err != nil {
		return err
	}
	if err := projectdata.WriteJSON(path, data, true); err != nil {
		return err
	}
	fmt.Printf("Assigned stable project IDs to %d catalog rows.\n", assigned)
	return nil
}

func validate(root, project string) error {
	catalog, err := projectdata.LoadData(filepath.Join(root, "awesome-agents.json"))
	if err != nil {
		return err
	}
	ledger, err := contributions.Read(filepath.Join(root, "contributions.json"))
	if err != nil {
		return err
	}
	if err := contributions.Validate(catalog, ledger); err != nil {
		return err
	}
	if project != "" {
		if err := contributions.RequireSubmitter(catalog, ledger, project); err != nil {
			return err
		}
	}
	fmt.Printf("Valid contribution ledger: %d catalog rows; %d credited projects.\n", len(catalog.Agents), len(ledger.Projects))
	return nil
}

func backfill(root string) error {
	repository := strings.TrimSpace(os.Getenv("GITHUB_REPOSITORY"))
	if repository == "" {
		repository = defaultRepository
	}
	parts := strings.Split(repository, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.ContainsAny(repository, "?# ") {
		return errors.New("GITHUB_REPOSITORY must be owner/repo")
	}
	catalog, err := projectdata.LoadData(filepath.Join(root, "awesome-agents.json"))
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	issues, err := fetchPages[issue](client, root, repository, "issues", "state=all")
	if err != nil {
		return err
	}
	pulls, err := fetchPages[pull](client, root, repository, "pulls", "state=all")
	if err != nil {
		return err
	}
	closedPulls := make([]pull, 0, len(pulls))
	for _, pr := range pulls {
		if pr.ClosedAt != "" {
			closedPulls = append(closedPulls, pr)
		}
	}
	fileCache, err := fetchClosedPullFiles(client, root, repository, closedPulls)
	if err != nil {
		return err
	}
	bySource := map[string][]projectdata.Project{}
	for _, p := range catalog.Agents {
		for _, source := range p.Sources {
			forge, path, _, ok := projectdata.ForgeRepository(source.SourceURL)
			if ok {
				key := forge + "/" + strings.ToLower(strings.TrimSuffix(path, ".git"))
				bySource[key] = append(bySource[key], p)
			}
		}
	}
	results := report{Repository: repository, GeneratedAt: time.Now().UTC().Format(time.RFC3339), PullRequestsScanned: len(pulls), ClosedPullsScanned: len(closedPulls), Candidates: []candidate{}, CatalogPulls: []catalogPullCandidate{}}
	seen := map[string]bool{}
	addedByProject := map[string][]string{}
	for _, pr := range closedPulls {
		files := fileCache[pr.Number]
		if !catalogFiles(files) {
			continue
		}
		addedProjects := make([]projectdata.Project, 0)
		matchedProjects := make([]projectdata.Project, 0)
		for _, p := range catalog.Agents {
			if pullMatchesProject(pr, files, p) {
				matchedProjects = append(matchedProjects, p)
			}
			if pullAddsProject(pr, files, p) {
				addedProjects = append(addedProjects, p)
				addedByProject[p.ID] = append(addedByProject[p.ID], pr.HTMLURL)
			}
		}
		if len(addedProjects) == 0 && strings.EqualFold(pr.User.Login, parts[0]) {
			continue
		}
		projectNames := func(items []projectdata.Project) []string {
			names := make([]string, 0, len(items))
			for _, item := range items {
				names = append(names, item.Project)
			}
			sort.Strings(names)
			return unique(names)
		}
		mergedAt := ""
		if pr.MergedAt != nil {
			mergedAt = *pr.MergedAt
		}
		fileNames := make([]string, 0, len(files))
		for _, file := range files {
			fileNames = append(fileNames, file.Filename)
		}
		results.CatalogPulls = append(results.CatalogPulls, catalogPullCandidate{
			Number: pr.Number, Title: pr.Title, URL: pr.HTMLURL, AuthorID: pr.User.ID, Author: pr.User.Login,
			MergedAt: mergedAt, ClosedAt: pr.ClosedAt, Files: fileNames,
			AddedProjects: projectNames(addedProjects), MatchedProjects: projectNames(matchedProjects),
		})
	}
	for _, item := range issues {
		if item.PullRequest != nil || item.State != "closed" {
			continue
		}
		results.ClosedIssuesScanned++
		matched := matchProjects(item.Body, bySource)
		for _, p := range matched {
			key := fmt.Sprintf("%s:%d", p.ID, item.Number)
			if seen[key] {
				continue
			}
			seen[key] = true
			candidate := candidate{ProjectID: p.ID, Project: p.Project, IssueNumber: item.Number, IssueURL: item.HTMLURL, AuthorID: item.User.ID, Author: item.User.Login, SubmittedAt: item.CreatedAt, FounderTeamClaim: founderClaim.MatchString(item.Body), MaintainerClaim: maintainerClaim.MatchString(item.Body)}
			for _, pr := range pulls {
				if pr.MergedAt == nil || !referencesIssue(pr.Body, item.Number) {
					continue
				}
				files := fileCache[pr.Number]
				if pullAddsProject(pr, files, p) {
					candidate.Accepted = true
					candidate.MergedPulls = append(candidate.MergedPulls, pr.HTMLURL)
				}
			}
			if !candidate.Accepted {
				for _, pr := range closedPulls {
					if pr.MergedAt == nil || !containsProject(addedByProject, p.ID, pr.HTMLURL) {
						continue
					}
					mergedAt, mergeErr := time.Parse(time.RFC3339, *pr.MergedAt)
					createdAt, createErr := time.Parse(time.RFC3339, item.CreatedAt)
					closedAt, closeErr := time.Parse(time.RFC3339, item.ClosedAt)
					if mergeErr == nil && createErr == nil && closeErr == nil && !mergedAt.Before(createdAt) && !mergedAt.After(closedAt) {
						candidate.RelatedCatalogPulls = append(candidate.RelatedCatalogPulls, pr.HTMLURL)
					}
				}
			}
			sort.Strings(candidate.MergedPulls)
			sort.Strings(candidate.RelatedCatalogPulls)
			results.Candidates = append(results.Candidates, candidate)
		}
	}
	for i := range results.Candidates {
		results.Candidates[i].MergedPulls = unique(results.Candidates[i].MergedPulls)
		results.Candidates[i].RelatedCatalogPulls = unique(results.Candidates[i].RelatedCatalogPulls)
	}
	sort.Slice(results.Candidates, func(i, j int) bool {
		if results.Candidates[i].IssueNumber == results.Candidates[j].IssueNumber {
			return results.Candidates[i].ProjectID < results.Candidates[j].ProjectID
		}
		return results.Candidates[i].IssueNumber < results.Candidates[j].IssueNumber
	})
	sort.Slice(results.CatalogPulls, func(i, j int) bool { return results.CatalogPulls[i].Number < results.CatalogPulls[j].Number })
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

func matchProjects(body string, projects map[string][]projectdata.Project) []projectdata.Project {
	matched := map[string]projectdata.Project{}
	for _, raw := range repositoryURLs.FindAllString(body, -1) {
		raw = strings.TrimRight(raw, ".,;:!?)]}")
		forge, path, _, ok := projectdata.ForgeRepository(raw)
		if !ok {
			continue
		}
		key := forge + "/" + strings.ToLower(strings.TrimSuffix(path, ".git"))
		for _, p := range projects[key] {
			matched[p.ID] = p
		}
	}
	out := make([]projectdata.Project, 0, len(matched))
	for _, p := range matched {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func fetchClosedPullFiles(client *http.Client, root, repository string, pulls []pull) (map[int][]pullFile, error) {
	type result struct {
		number int
		files  []pullFile
		err    error
	}
	jobs := make(chan pull)
	results := make(chan result, len(pulls))
	var workers sync.WaitGroup
	workerCount := 6
	if len(pulls) < workerCount {
		workerCount = len(pulls)
	}
	for i := 0; i < workerCount; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for pr := range jobs {
				files, err := fetchPullFiles(client, root, repository, pr.Number)
				results <- result{number: pr.Number, files: files, err: err}
			}
		}()
	}
	for _, pr := range pulls {
		jobs <- pr
	}
	close(jobs)
	workers.Wait()
	close(results)
	fileCache := make(map[int][]pullFile, len(pulls))
	for result := range results {
		if result.err != nil {
			return nil, fmt.Errorf("fetch pull request #%d files: %w", result.number, result.err)
		}
		fileCache[result.number] = result.files
	}
	return fileCache, nil
}

func fetchPages[T any](client *http.Client, root, repository, endpoint, query string) ([]T, error) {
	var result []T
	for page := 1; ; page++ {
		path := fmt.Sprintf("https://api.github.com/repos/%s/%s?per_page=100&page=%d", repository, endpoint, page)
		if query != "" {
			path += "&" + query
		}
		var batch []T
		if err := getJSON(client, root, path, &batch); err != nil {
			return nil, err
		}
		result = append(result, batch...)
		if len(batch) < 100 {
			return result, nil
		}
	}
}

func fetchPullFiles(client *http.Client, root, repository string, number int) ([]pullFile, error) {
	files, err := fetchPages[pullFile](client, root, repository, fmt.Sprintf("pulls/%d/files", number), "")
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Filename < files[j].Filename })
	return files, nil
}

func catalogFiles(files []pullFile) bool {
	for _, file := range files {
		switch strings.ToLower(filepath.Base(file.Filename)) {
		case "awesome-agents.json", "awesome-agents.yaml", "awesome-agents.yml", "readme.md":
			return true
		}
	}
	return false
}

func pullMatchesProject(pr pull, files []pullFile, project projectdata.Project) bool {
	if !catalogFiles(files) {
		return false
	}
	var content strings.Builder
	content.WriteString(strings.ToLower(pr.Title + "\n" + pr.Body))
	for _, file := range files {
		content.WriteString("\n")
		content.WriteString(strings.ToLower(file.Patch))
	}
	text := content.String()
	if strings.Contains(text, strings.ToLower(project.Project)) {
		return true
	}
	for _, source := range project.Sources {
		_, _, canonical, ok := projectdata.ForgeRepository(source.SourceURL)
		if ok && strings.Contains(text, strings.ToLower(canonical)) {
			return true
		}
	}
	return false
}

func pullAddsProject(pr pull, files []pullFile, project projectdata.Project) bool {
	if !catalogFiles(files) {
		return false
	}
	name := strings.ToLower(strings.TrimSpace(project.Project))
	canonicalSources := make([]string, 0, len(project.Sources))
	for _, source := range project.Sources {
		_, _, canonical, ok := projectdata.ForgeRepository(source.SourceURL)
		if ok {
			canonicalSources = append(canonicalSources, strings.ToLower(canonical))
		}
	}
	for _, file := range files {
		var addedName, addedSource bool
		for _, line := range strings.Split(file.Patch, "\n") {
			if !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
				continue
			}
			line = strings.ToLower(line[1:])
			if (strings.Contains(line, "\"project\"") || strings.Contains(line, "project:")) && strings.Contains(line, name) {
				addedName = true
			}
			if strings.Contains(line, "**"+name+"**") || strings.Contains(line, "["+name+"](") {
				addedName = true
			}
			if strings.TrimSpace(strings.TrimLeft(line, "# ")) == name && strings.HasPrefix(strings.TrimSpace(line), "### ") {
				addedName = true
			}
			for _, source := range canonicalSources {
				if strings.Contains(line, source) {
					addedSource = true
				}
			}
		}
		if addedName && addedSource {
			return true
		}
	}
	return false
}

func containsProject(projectPulls map[string][]string, projectID, pullURL string) bool {
	for _, candidate := range projectPulls[projectID] {
		if candidate == pullURL {
			return true
		}
	}
	return false
}

func unique(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func referencesIssue(body string, number int) bool {
	wanted := fmt.Sprint(number)
	for _, match := range issueReference.FindAllStringSubmatch(body, -1) {
		if match[1] == wanted {
			return true
		}
	}
	for _, match := range issueURLReference.FindAllStringSubmatch(body, -1) {
		if match[1] == wanted {
			return true
		}
	}
	return false
}

func getJSON(client *http.Client, root, endpoint string, target any) error {
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "awesome-ai-agents-contribution-backfill/1.0")
	if token := projectdata.GitHubToken(root); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("GitHub API HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(response.Body).Decode(target)
}
