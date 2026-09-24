// Command verify-forge-repositories confirms that cataloged third-party projects
// have a public repository on GitHub, GitLab.com, or Codeberg. License presence
// or category is deliberately not evaluated.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"awesome-ai-agents/internal/forgeverify"
	"awesome-ai-agents/internal/projectdata"
)

func main() {
	project := flag.String("project", "", "verify only this exact project name")
	all := flag.Bool("all", false, "verify every project (explicit full-catalog audit)")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "ERROR: unexpected positional arguments")
		os.Exit(2)
	}
	projectName := strings.TrimSpace(*project)
	if (projectName != "") == *all {
		fmt.Fprintln(os.Stderr, "ERROR: specify exactly one of --project <name> or --all")
		os.Exit(2)
	}
	if err := run(projectName, *all); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}

func run(projectName string, fullAudit bool) error {
	root := projectdata.Root()
	var data projectdata.Data
	if err := projectdata.ReadJSON(root+"/awesome-agents.json", &data); err != nil {
		return err
	}
	projectIndexes := make([]int, 0, len(data.Agents))
	if fullAudit {
		for i := range data.Agents {
			projectIndexes = append(projectIndexes, i)
		}
	} else {
		for i := range data.Agents {
			if data.Agents[i].Project == projectName {
				projectIndexes = append(projectIndexes, i)
			}
		}
		if len(projectIndexes) == 0 {
			return fmt.Errorf("project %q not found", projectName)
		}
		if len(projectIndexes) > 1 {
			return fmt.Errorf("project name %q is not unique", projectName)
		}
	}
	client := &http.Client{Timeout: 30 * time.Second}
	failures, checked, ineligible := 0, 0, 0
	for _, i := range projectIndexes {
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
	if !fullAudit {
		fmt.Printf("Forge checks: %d; project checked: %s; qualifying repository: %t; API failures: %d\n", checked, projectName, ineligible == 0, failures)
		if ineligible > 0 {
			return fmt.Errorf("project %q has no verified public repository", projectName)
		}
	} else {
		fmt.Printf("Forge checks: %d; projects without a qualifying repository: %d; API failures: %d\n", checked, ineligible, failures)
	}
	return nil
}
