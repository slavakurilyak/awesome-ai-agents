package projectdata

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"
)

type Source struct {
	Source                 string  `json:"source" yaml:"source"`
	SourceURL              string  `json:"source_url" yaml:"source_url"`
	RepositoryOwner        string  `json:"repository_owner,omitempty" yaml:"repository_owner,omitempty"`
	RepositoryOwnerURL     string  `json:"repository_owner_url,omitempty" yaml:"repository_owner_url,omitempty"`
	Stars                  *int    `json:"stars,omitempty" yaml:"stars,omitempty"`
	StarsLastUpdated       *string `json:"stars_last_updated" yaml:"stars_last_updated"`
	Badge                  string  `json:"badge,omitempty" yaml:"badge,omitempty"`
	RepositoryCheckedAt    string  `json:"repository_checked_at,omitempty"`
	RepositoryStatus       string  `json:"repository_status,omitempty"`
	RepositoryStatusDetail string  `json:"repository_status_detail,omitempty"`
}
type Project struct {
	ID                  string   `json:"id" yaml:"id"`
	Project             string   `json:"project" yaml:"project"`
	ProjectDescription  *string  `json:"project_description,omitempty" yaml:"project_description"`
	HasPublicRepository bool     `json:"has_public_repository" yaml:"has_public_repository"`
	Categories          []string `json:"categories" yaml:"categories"`
	Capabilities        []string `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	Interfaces          []string `json:"interfaces,omitempty" yaml:"interfaces,omitempty"`
	Sources             []Source `json:"sources" yaml:"sources"`
}

// UnmarshalJSON accepts the retired project_is_open_source key so submissions
// prepared before the rename to has_public_repository still validate.
func (p *Project) UnmarshalJSON(b []byte) error {
	type plain Project
	aux := struct {
		*plain
		LegacyIsOpenSource bool `json:"project_is_open_source"`
	}{plain: (*plain)(p)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	p.HasPublicRepository = p.HasPublicRepository || aux.LegacyIsOpenSource
	return nil
}

// NewProjectID creates a stable opaque key from the initial project name and
// primary repository. Store it in the catalog once and preserve it on renames
// and repository transfers.
func NewProjectID(p Project) string {
	var slug strings.Builder
	separator := false
	for _, r := range strings.ToLower(strings.TrimSpace(p.Project)) {
		if unicode.IsLetter(r) && r <= unicode.MaxASCII || r >= '0' && r <= '9' {
			slug.WriteRune(r)
			separator = false
		} else if slug.Len() > 0 && !separator {
			slug.WriteByte('-')
			separator = true
		}
	}
	key := strings.Trim(slug.String(), "-")
	if key == "" {
		key = "project"
	}
	primary := ""
	if len(p.Sources) > 0 {
		_, _, primary, _ = ForgeRepository(p.Sources[0].SourceURL)
	}
	seed := strings.ToLower(strings.TrimSpace(p.Project)) + "\x00" + strings.ToLower(primary)
	sum := sha256.Sum256([]byte(seed))
	return key + "-" + hex.EncodeToString(sum[:4])
}

func EnsureProjectID(p *Project) bool {
	if p.ID != "" {
		return false
	}
	p.ID = NewProjectID(*p)
	return true
}

type Category struct {
	Category            string `json:"category" yaml:"category"`
	CategoryDescription string `json:"category_description" yaml:"category_description"`
}
type Data struct {
	Agents     []Project  `json:"agents"`
	Categories []Category `json:"categories"`
}
type HistoryRow struct {
	Repository string `json:"repository"`
	Date       string `json:"date"`
	Stars      *int   `json:"stars,omitempty"`
	Status     string `json:"status,omitempty"`
}
type History struct {
	Snapshots []HistoryRow `json:"snapshots"`
}

func Root() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// GitHubToken prefers the process environment and falls back to the repository
// .env file for local scheduled runs. It never logs the credential.
func GitHubToken(root string) string {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	file, err := os.Open(filepath.Join(root, ".env"))
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "GITHUB_TOKEN" {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		return value
	}
	return ""
}

func ReadYAML(path string, target any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	return yaml.Unmarshal(b, target)
}
func ReadJSON(path string, target any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	if e = json.Unmarshal(b, target); e != nil {
		return fmt.Errorf("decode %s: %w", path, e)
	}
	return nil
}
func WriteJSON(path string, v any, newline bool) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	if newline {
		b = append(b, '\n')
	}
	return os.WriteFile(path, b, 0644)
}
func LoadData(path string) (Data, error) {
	var d Data
	e := ReadJSON(path, &d)
	if e != nil {
		return d, e
	}
	return d, Validate(d)
}

// Validate checks the fields required for a project to be reviewed and rendered.
func Validate(d Data) error {
	if len(d.Agents) == 0 {
		return fmt.Errorf("agents must contain at least one project")
	}
	ids := make(map[string]string, len(d.Agents))
	for i, p := range d.Agents {
		if strings.TrimSpace(p.ID) == "" {
			return fmt.Errorf("agents[%d] (%s).id is required", i, p.Project)
		}
		if strings.ContainsAny(p.ID, " /\\?#") {
			return fmt.Errorf("agents[%d] (%s).id must be a stable URL-safe identifier", i, p.Project)
		}
		identity := strings.ToLower(strings.TrimSpace(p.Project))
		if len(p.Sources) > 0 {
			_, _, canonical, _ := ForgeRepository(p.Sources[0].SourceURL)
			identity += "\x00" + strings.ToLower(canonical)
		}
		if old, exists := ids[strings.ToLower(p.ID)]; exists && old != identity {
			return fmt.Errorf("agents[%d] (%s).id %q conflicts with another project", i, p.Project, p.ID)
		}
		ids[strings.ToLower(p.ID)] = identity
		if strings.TrimSpace(p.Project) == "" {
			return fmt.Errorf("agents[%d].project is required", i)
		}
		if p.ProjectDescription == nil || strings.TrimSpace(*p.ProjectDescription) == "" {
			return fmt.Errorf("agents[%d] (%s).project_description is required", i, p.Project)
		}
		if len(p.Categories) == 0 {
			return fmt.Errorf("agents[%d] (%s).categories must contain at least one category", i, p.Project)
		}
		if len(p.Sources) == 0 {
			return fmt.Errorf("agents[%d] (%s).sources must contain at least one verifiable URL", i, p.Project)
		}
		if !p.HasPublicRepository {
			return fmt.Errorf("agents[%d] (%s) is not eligible: every listed project must have a public GitHub, GitLab, or Codeberg repository", i, p.Project)
		}
		verified := false
		for _, source := range p.Sources {
			if _, _, _, ok := ForgeRepository(source.SourceURL); !ok {
				continue
			}
			if source.RepositoryStatus != "active" && source.RepositoryStatus != "archived" {
				continue
			}
			if _, err := time.Parse(time.RFC3339, source.RepositoryCheckedAt); err == nil {
				verified = true
				break
			}
		}
		if !verified {
			return fmt.Errorf("agents[%d] (%s) needs a verified public repository on GitHub, GitLab, or Codeberg", i, p.Project)
		}
		for j, source := range p.Sources {
			parsed, err := url.ParseRequestURI(strings.TrimSpace(source.SourceURL))
			if strings.TrimSpace(source.Source) == "" || err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
				return fmt.Errorf("agents[%d] (%s).sources[%d] requires a source label and an http(s) URL", i, p.Project, j)
			}
		}
	}
	return nil
}

// ForgeRepository returns the forge, API repository path, and canonical URL
// for direct repository links. Only the three supported public forges qualify.
func ForgeRepository(raw string) (forge, path, canonical string, ok bool) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return "", "", "", false
	}
	host := strings.ToLower(strings.TrimPrefix(u.Hostname(), "www."))
	parts := strings.FieldsFunc(strings.Trim(u.Path, "/"), func(r rune) bool { return r == '/' })
	if len(parts) < 2 {
		return "", "", "", false
	}
	forge = host
	switch host {
	case "github.com":
		forge = "github"
		path = strings.Join(parts[:2], "/")
		canonical = "https://" + host + "/" + path
	case "codeberg.org":
		forge = "codeberg"
		path = strings.Join(parts[:2], "/")
		canonical = "https://" + host + "/" + path
	case "gitlab.com":
		forge = "gitlab"
		end := len(parts)
		for i, part := range parts {
			if part == "-" || part == "tree" || part == "blob" || part == "issues" || part == "merge_requests" || part == "pipelines" {
				end = i
				break
			}
		}
		if end < 2 {
			return "", "", "", false
		}
		path = strings.Join(parts[:end], "/")
		canonical = "https://gitlab.com/" + path
	default:
		return "", "", "", false
	}
	path = strings.TrimSuffix(path, ".git")
	canonical = strings.TrimSuffix(canonical, ".git")
	return forge, path, canonical, true
}
