package forgeverify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"awesome-ai-agents/internal/projectdata"
)

// Repository checks public forge metadata. No license is required by this list's policy.
func Repository(client *http.Client, root, forge, repo string) (public bool, stars int, status, owner, ownerURL string, err error) {
	endpoint := ""
	switch forge {
	case "github":
		endpoint = "https://api.github.com/repos/" + repo
	case "gitlab":
		endpoint = "https://gitlab.com/api/v4/projects/" + url.PathEscape(repo)
	case "codeberg":
		endpoint = "https://codeberg.org/api/v1/repos/" + repo
	default:
		return false, -1, "", "", "", fmt.Errorf("unsupported forge %q", forge)
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return false, -1, "", "", "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "awesome-ai-agents-forge-verifier/1.0")
	if forge == "github" {
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if token := projectdata.GitHubToken(root); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, -1, "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return false, -1, "not_found", "", "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, -1, "", "", "", fmt.Errorf("%s API returned HTTP %d", forge, resp.StatusCode)
	}
	var metadata struct {
		Private           bool   `json:"private"`
		Archived          bool   `json:"archived"`
		Visibility        string `json:"visibility"`
		Stars             int    `json:"stargazers_count"`
		GitLabStars       int    `json:"star_count"`
		FullName          string `json:"full_name"`
		PathWithNamespace string `json:"path_with_namespace"`
		Owner             struct {
			Login    string `json:"login"`
			HTMLURL  string `json:"html_url"`
			FullPath string `json:"full_path"`
			WebURL   string `json:"web_url"`
		} `json:"owner"`
		Namespace struct {
			FullPath string `json:"full_path"`
			WebURL   string `json:"web_url"`
		} `json:"namespace"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return false, -1, "", "", "", err
	}
	owner = metadata.Owner.Login
	ownerURL = metadata.Owner.HTMLURL
	if forge == "gitlab" {
		owner = metadata.Namespace.FullPath
		ownerURL = metadata.Namespace.WebURL
		if owner == "" && metadata.PathWithNamespace != "" {
			owner = strings.TrimSuffix(metadata.PathWithNamespace, "/"+pathBase(repo))
			ownerURL = "https://gitlab.com/" + owner
		}
	}
	if forge == "github" && owner == "" && metadata.FullName != "" {
		owner = strings.SplitN(metadata.FullName, "/", 2)[0]
		ownerURL = "https://github.com/" + owner
	}
	if forge == "codeberg" && ownerURL == "" && owner != "" {
		ownerURL = "https://codeberg.org/" + owner
	}
	if metadata.Private || (metadata.Visibility != "" && !strings.EqualFold(metadata.Visibility, "public")) {
		return false, -1, "private", owner, ownerURL, nil
	}
	if forge == "gitlab" {
		metadata.Stars = metadata.GitLabStars
	}
	status = "active"
	if metadata.Archived {
		status = "archived"
	}
	return true, metadata.Stars, status, owner, ownerURL, nil
}

func pathBase(repo string) string {
	parts := strings.Split(strings.Trim(repo, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
