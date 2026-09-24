// Command validate-data validates the canonical project dataset before a PR is submitted.
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
	root := projectdata.Root()
	data, err := projectdata.LoadData(root + "/awesome-agents.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	verified := 0
	for _, project := range data.Agents {
		found := false
		for _, source := range project.Sources {
			forge, repo, _, ok := projectdata.ForgeRepository(source.SourceURL)
			if !ok {
				continue
			}
			public, _, _, checkErr := forgeverify.Repository(client, root, forge, repo)
			if checkErr != nil {
				fmt.Fprintf(os.Stderr, "ERROR: verify %s on %s: %v\n", project.Project, forge, checkErr)
				os.Exit(1)
			}
			if public {
				found = true
				verified++
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "ERROR: %s has no verifiable public repository on GitHub, GitLab.com, or Codeberg\n", project.Project)
			os.Exit(1)
		}
	}
	fmt.Printf("Valid project data: %d projects; live public-forge checks passed for all %d projects\n", len(data.Agents), verified)
}
