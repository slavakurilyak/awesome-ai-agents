// Command verify-forge-repositories confirms that cataloged third-party projects
// have a public repository on GitHub, GitLab.com, or Codeberg. License presence
// or category is deliberately not evaluated.
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"awesome-ai-agents/internal/forgeverify"
	"awesome-ai-agents/internal/projectdata"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}

func run() error {
	root := projectdata.Root()
	var data projectdata.Data
	if err := projectdata.ReadJSON(root+"/awesome-agents.json", &data); err != nil {
		return err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	failures, checked, ineligible := 0, 0, 0
	for i := range data.Agents {
		project := &data.Agents[i]
		found := false
		for j := range project.Sources {
			source := &project.Sources[j]
			forge, repo, canonical, ok := projectdata.ForgeRepository(source.SourceURL)
			if !ok {
				continue
			}
			checked++
			at := time.Now().UTC().Format(time.RFC3339)
			public, _, status, err := forgeverify.Repository(client, root, forge, repo)
			source.RepositoryCheckedAt = at
			if err != nil {
				failures++
				source.RepositoryStatus = "check_failed"
				source.RepositoryStatusDetail = err.Error()
				fmt.Fprintf(os.Stderr, "WARN %s (%s): %v\n", project.Project, canonical, err)
				continue
			}
			source.Source = forge
			source.RepositoryStatusDetail = ""
			if !public {
				source.RepositoryStatus = status
				continue
			}
			source.RepositoryStatus = "active"
			if status == "archived" {
				source.RepositoryStatus = "archived"
			}
			project.ProjectIsOpenSource = true
			found = true
			break
		}
		if !found {
			project.ProjectIsOpenSource = false
			ineligible++
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d forge checks failed; do not treat failed checks as proof of eligibility", failures)
	}
	if err := projectdata.WriteJSON(root+"/awesome-agents.json", data, true); err != nil {
		return err
	}
	fmt.Printf("Forge checks: %d; projects without a qualifying repository: %d; API failures: %d\n", checked, ineligible, failures)
	return nil
}
