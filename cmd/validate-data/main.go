// Command validate-data validates the canonical project dataset before a PR is submitted.
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
	projectName := flag.String("project", "", "live-check only this exact project name")
	all := flag.Bool("all", false, "live-check every project (explicit full-catalog audit)")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "ERROR: unexpected positional arguments")
		os.Exit(2)
	}
	name := strings.TrimSpace(*projectName)
	if (name != "") == *all {
		fmt.Fprintln(os.Stderr, "ERROR: specify exactly one of --project <name> or --all")
		os.Exit(2)
	}
	root := projectdata.Root()
	data, err := projectdata.LoadData(root + "/awesome-agents.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
	projectIndexes := make([]int, 0, len(data.Agents))
	if *all {
		for i := range data.Agents {
			projectIndexes = append(projectIndexes, i)
		}
	} else {
		for i := range data.Agents {
			if data.Agents[i].Project == name {
				projectIndexes = append(projectIndexes, i)
			}
		}
		if len(projectIndexes) == 0 {
			fmt.Fprintf(os.Stderr, "ERROR: project %q not found\n", name)
			os.Exit(1)
		}
		if len(projectIndexes) > 1 {
			fmt.Fprintf(os.Stderr, "ERROR: project name %q is not unique\n", name)
			os.Exit(1)
		}
	}
	client := &http.Client{Timeout: 30 * time.Second}
	verified := 0
	for _, i := range projectIndexes {
		project := data.Agents[i]
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
	if !*all {
		fmt.Printf("Valid project data: %d projects structurally valid; live public-forge check passed for %q (%d repository checks)\n", len(data.Agents), name, verified)
	} else {
		fmt.Printf("Valid project data: %d projects; live public-forge checks passed for all %d projects\n", len(data.Agents), verified)
	}
}
