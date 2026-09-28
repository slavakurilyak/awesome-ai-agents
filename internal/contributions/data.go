package contributions

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"awesome-ai-agents/internal/projectdata"
)

type Credit struct {
	GitHubID               int64    `json:"github_id"`
	Login                  string   `json:"login"`
	EvidenceURL            string   `json:"evidence_url"`
	AdditionalEvidenceURLs []string `json:"additional_evidence_urls,omitempty"`
	AcceptanceURL          string   `json:"acceptance_url,omitempty"`
	At                     string   `json:"at"`
	Status                 string   `json:"status"`
	FounderTeamClaim       bool     `json:"founder_team_claim,omitempty"`
}

type Project struct {
	ProjectID    string   `json:"project_id"`
	SubmittedBy  []Credit `json:"submitted_by,omitempty"`
	MaintainedBy []Credit `json:"maintained_by,omitempty"`
}

type Data struct {
	Projects []Project `json:"projects"`
}

var githubLogin = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37})$`)

func Read(path string) (Data, error) {
	var d Data
	if err := projectdata.ReadJSON(path, &d); err != nil {
		return d, err
	}
	return d, nil
}

func Write(path string, d Data) error {
	return projectdata.WriteJSON(path, d, true)
}

func Validate(catalog projectdata.Data, data Data) error {
	projects := make(map[string]bool, len(catalog.Agents))
	for _, p := range catalog.Agents {
		projects[p.ID] = true
	}
	seen := make(map[string]bool, len(data.Projects))
	for i, project := range data.Projects {
		if project.ProjectID == "" || !projects[project.ProjectID] {
			return fmt.Errorf("projects[%d].project_id %q does not identify a catalog project", i, project.ProjectID)
		}
		if seen[project.ProjectID] {
			return fmt.Errorf("projects[%d].project_id %q is duplicated", i, project.ProjectID)
		}
		seen[project.ProjectID] = true
		for j, credit := range project.SubmittedBy {
			if err := validateCredit(credit); err != nil {
				return fmt.Errorf("projects[%d].submitted_by[%d]: %w", i, j, err)
			}
			if credit.Status != "accepted_submission" {
				return fmt.Errorf("projects[%d].submitted_by[%d].status must be accepted_submission", i, j)
			}
			if !repositoryIssueURL(credit.EvidenceURL) {
				return fmt.Errorf("projects[%d].submitted_by[%d].evidence_url must link to an issue or pull request in this repository", i, j)
			}
			if !repositoryPullURL(credit.AcceptanceURL) {
				return fmt.Errorf("projects[%d].submitted_by[%d].acceptance_url must link to the accepted pull request in this repository", i, j)
			}
			for k, evidenceURL := range credit.AdditionalEvidenceURLs {
				if !repositoryIssueURL(evidenceURL) {
					return fmt.Errorf("projects[%d].submitted_by[%d].additional_evidence_urls[%d] must link to an issue or pull request in this repository", i, j, k)
				}
			}
		}
		for j, credit := range project.MaintainedBy {
			if err := validateCredit(credit); err != nil {
				return fmt.Errorf("projects[%d].maintained_by[%d]: %w", i, j, err)
			}
			if credit.Status != "self_reported" && credit.Status != "source_confirmed" {
				return fmt.Errorf("projects[%d].maintained_by[%d].status must be self_reported or source_confirmed", i, j)
			}
			if !httpsURL(credit.EvidenceURL) {
				return fmt.Errorf("projects[%d].maintained_by[%d].evidence_url must be an https URL", i, j)
			}
		}
	}
	return nil
}

func validateCredit(c Credit) error {
	if c.GitHubID <= 0 {
		return fmt.Errorf("github_id must be positive")
	}
	if !githubLogin.MatchString(c.Login) {
		return fmt.Errorf("login %q is not a valid GitHub login", c.Login)
	}
	if !httpsURL(c.EvidenceURL) {
		return fmt.Errorf("evidence_url must be an https URL")
	}
	if _, err := time.Parse(time.RFC3339, c.At); err != nil {
		return fmt.Errorf("at must be an RFC3339 timestamp")
	}
	return nil
}

func repositoryIssueURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, "https") || !strings.EqualFold(u.Hostname(), "github.com") || u.User != nil {
		return false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	return len(parts) == 4 && strings.EqualFold(parts[0], "slavakurilyak") && strings.EqualFold(parts[1], "awesome-ai-agents") && (parts[2] == "issues" || parts[2] == "pull")
}

func repositoryPullURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(u.Scheme, "https") || !strings.EqualFold(u.Hostname(), "github.com") || u.User != nil {
		return false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	return len(parts) == 4 && strings.EqualFold(parts[0], "slavakurilyak") && strings.EqualFold(parts[1], "awesome-ai-agents") && parts[2] == "pull"
}

func httpsURL(raw string) bool {
	u, err := url.ParseRequestURI(strings.TrimSpace(raw))
	return err == nil && strings.EqualFold(u.Scheme, "https") && u.Hostname() != "" && u.User == nil
}

func Decode(b []byte) (Data, error) {
	var d Data
	err := json.Unmarshal(b, &d)
	return d, err
}
