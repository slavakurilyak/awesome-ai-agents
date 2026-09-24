package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"awesome-ai-agents/internal/projectdata"
	"gopkg.in/yaml.v3"
)

type item struct {
	Project projectdata.Project
	Growth  *growth
}
type growth struct {
	Delta, Days int
	Relative    float64
}
type emojiFile struct {
	CategoryEmojis map[string]string `yaml:"category_emojis"`
}

var githubPattern = regexp.MustCompile(`(?i)github\.com/([^/]+)/([^/#?]+)`)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", e)
		os.Exit(1)
	}
}
func run() error {
	root := projectdata.Root()
	data, e := projectdata.LoadData(root + "/awesome-agents.json")
	if e != nil {
		return e
	}
	emojis, e := loadEmojis(root + "/awesome-categories.yaml")
	if e != nil {
		fmt.Fprintln(os.Stderr, "WARN:", e)
	}
	template, e := os.ReadFile(root + "/README.template.md")
	if e != nil {
		return e
	}
	hist := loadHistory(root + "/github-stars-history.json")
	sections := renderSections(data, emojis, hist)
	top, names := topProjects(data)
	rising := risingProjects(data, names, hist)
	risingContent := renderList(rising)
	if len(rising) == 0 {
		if hasGrowthBaseline(data, hist) {
			risingContent = "<p><em>No positive seven-day star growth found among eligible projects.</em></p>"
		} else {
			risingContent = "<p><em>Collecting daily snapshots. Rising projects will appear once seven-day growth can be calculated.</em></p>"
		}
	}
	content := strings.ReplaceAll(string(template), "${SECTIONS}", sections)
	content = strings.ReplaceAll(content, "${TOP_STARRED_PROJECTS}", renderList(top))
	content = strings.ReplaceAll(content, "${RISING_PROJECTS}", risingContent)
	if e = os.WriteFile(root+"/README.md", []byte(content), 0644); e != nil {
		return e
	}
	fmt.Printf("Successfully generated %s\n", root+"/README.md")
	return nil
}

func hasGrowthBaseline(data projectdata.Data, history map[string][]projectdata.HistoryRow) bool {
	for _, project := range data.Agents {
		if projectGrowth(project, history, 7) != nil {
			return true
		}
	}
	return false
}

func loadEmojis(path string) (map[string]string, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	var raw any
	if e = yaml.Unmarshal(b, &raw); e != nil {
		return nil, e
	}
	out := map[string]string{}
	switch v := raw.(type) {
	case []any:
		for _, x := range v {
			if m, ok := x.(map[string]any); ok {
				c, _ := m["category"].(string)
				em, _ := m["emoji"].(string)
				if c != "" && em != "" {
					out[c] = em
				}
			}
		}
	case map[string]any:
		if m, ok := v["category_emojis"].(map[string]any); ok {
			for k, x := range m {
				if s, ok := x.(string); ok {
					out[k] = s
				}
			}
		}
	}
	return out, nil
}
func loadHistory(path string) map[string][]projectdata.HistoryRow {
	var h projectdata.History
	if projectdata.ReadJSON(path, &h) != nil {
		return nil
	}
	m := map[string][]projectdata.HistoryRow{}
	for _, r := range h.Snapshots {
		if r.Stars != nil && r.Status == "active" {
			k := strings.ToLower(r.Repository)
			m[k] = append(m[k], r)
		}
	}
	for k := range m {
		sort.Slice(m[k], func(i, j int) bool { return m[k][i].Date < m[k][j].Date })
	}
	return m
}
func github(p projectdata.Project) *projectdata.Source {
	for i := range p.Sources {
		if p.Sources[i].Source == "github" {
			return &p.Sources[i]
		}
	}
	return nil
}
func stars(p projectdata.Project) int {
	s := github(p)
	if s == nil || s.Stars == nil {
		return -1
	}
	return *s.Stars
}
func repoKey(url string) string {
	m := githubPattern.FindStringSubmatch(url)
	if len(m) < 3 {
		return ""
	}
	return strings.ToLower(m[1] + "/" + strings.TrimSuffix(m[2], ".git"))
}
func projectGrowth(p projectdata.Project, h map[string][]projectdata.HistoryRow, days int) *growth {
	s := github(p)
	if s == nil {
		return nil
	}
	rows := h[repoKey(s.SourceURL)]
	if len(rows) < 2 {
		return nil
	}
	latest, e := time.Parse("2006-01-02", rows[len(rows)-1].Date)
	if e != nil {
		return nil
	}
	target := latest.AddDate(0, 0, -days)
	base := -1
	for i := 0; i < len(rows)-1; i++ {
		d, e := time.Parse("2006-01-02", rows[i].Date)
		if e == nil && !d.After(target) {
			base = i
		}
	}
	if base < 0 {
		return nil
	}
	prev, e := time.Parse("2006-01-02", rows[base].Date)
	if e != nil {
		return nil
	}
	elapsed := int(latest.Sub(prev).Hours() / 24)
	if elapsed > days+2 {
		return nil
	}
	delta := *rows[len(rows)-1].Stars - *rows[base].Stars
	relative := 0.0
	if *rows[base].Stars != 0 {
		relative = float64(delta) / float64(*rows[base].Stars) * 100
	}
	return &growth{delta, elapsed, relative}
}
func topProjects(d projectdata.Data) ([]item, map[string]bool) {
	all := []item{}
	for _, p := range d.Agents {
		if stars(p) >= 0 && github(p) != nil {
			all = append(all, item{Project: p})
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return stars(all[i].Project) > stars(all[j].Project) })
	if len(all) > 10 {
		all = all[:10]
	}
	exclude := map[string]bool{}
	for _, x := range all {
		exclude[x.Project.Project] = true
	}
	return all, exclude
}
func risingProjects(d projectdata.Data, exclude map[string]bool, h map[string][]projectdata.HistoryRow) []item {
	all := []item{}
	for _, p := range d.Agents {
		if exclude[p.Project] || stars(p) < 100 {
			continue
		}
		g := projectGrowth(p, h, 7)
		if g != nil && g.Delta > 0 {
			all = append(all, item{p, g})
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i].Growth, all[j].Growth
		if a.Relative != b.Relative {
			return a.Relative > b.Relative
		}
		return a.Delta > b.Delta
	})
	if len(all) > 10 {
		all = all[:10]
	}
	return all
}
func dateText(s *string) string {
	if s == nil {
		return ""
	}
	v := *s
	if t, e := time.Parse(time.RFC3339, v); e == nil {
		return t.Format("2006-01-02")
	}
	if len(v) >= 10 {
		return v[:10]
	}
	return v
}
func renderList(items []item) string {
	if len(items) == 0 {
		return "<p><em>No projects to display.</em></p>"
	}
	var b strings.Builder
	b.WriteString("<ol>\n")
	for _, x := range items {
		p := x.Project
		src := github(p)
		if src == nil {
			src = &p.Sources[0]
		}
		display := ""
		if src.Stars != nil {
			display = fmt.Sprintf(" - %s stars", comma(*src.Stars))
			if d := dateText(src.StarsLastUpdated); d != "" {
				display += " (Updated: " + d + ")"
			}
		}
		if x.Growth != nil {
			display += fmt.Sprintf(" · %s stars / %dd (%+.1f%%)", signedComma(x.Growth.Delta), x.Growth.Days, x.Growth.Relative)
		}
		desc := ""
		if p.ProjectDescription != nil {
			desc = "<br>" + *p.ProjectDescription
		}
		fmt.Fprintf(&b, "<li><a href=\"%s\"><strong>%s</strong></a>%s%s</li>\n", src.SourceURL, p.Project, display, desc)
	}
	b.WriteString("</ol>")
	return b.String()
}
func renderSections(d projectdata.Data, em map[string]string, h map[string][]projectdata.HistoryRow) string {
	type agg struct {
		p    projectdata.Project
		cats map[string]bool
	}
	m := map[string]*agg{}
	for _, p := range d.Agents {
		k := strings.ToLower(p.Project)
		if m[k] == nil {
			m[k] = &agg{p, map[string]bool{}}
		}
		for _, c := range p.Categories {
			m[k].cats[c] = true
		}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out strings.Builder
	for _, k := range keys {
		a := m[k]
		p := a.p
		badge := p.Sources[0].SourceURL
		if s := github(p); s != nil {
			badge = s.SourceURL
		}
		yn, color := "Unverified", "red"
		if p.ProjectIsOpenSource {
			yn, color = "Public", "green"
		}
		starBadge := ""
		if s := github(p); s != nil {
			if mm := githubPattern.FindStringSubmatch(s.SourceURL); len(mm) > 2 {
				starBadge = fmt.Sprintf("<a href=\"%s\"><img src=\"https://img.shields.io/github/stars/%s/%s?style=social\" alt=\"GitHub stars\"></a>", s.SourceURL, mm[1], mm[2])
			}
		}
		fmt.Fprintf(&out, "### %s\n<div><a href=\"%s\"><img src=\"https://img.shields.io/badge/Repository-%s-%s\" alt=\"Repository verification\"></a> %s</div>\n", p.Project, badge, yn, color, starBadge)
		if s := github(p); s != nil && s.Stars != nil {
			date := dateText(s.StarsLastUpdated)
			growthLabels := []string{}
			for _, window := range []int{7, 30} {
				if g := projectGrowth(p, h, window); g != nil {
					growthLabels = append(growthLabels, fmt.Sprintf("%dd %s (%+.1f%%)", window, signedComma(g.Delta), g.Relative))
				}
			}
			fmt.Fprintf(&out, "<p>⭐ %s stars", comma(*s.Stars))
			if date != "" {
				fmt.Fprintf(&out, " (Updated: %s)", date)
			}
			if len(growthLabels) > 0 {
				fmt.Fprintf(&out, " · Growth: %s", strings.Join(growthLabels, " · "))
			}
			fmt.Fprintln(&out, "</p>")
		}
		cats := []string{}
		for c := range a.cats {
			cats = append(cats, c)
		}
		sort.Strings(cats)
		shown := []string{}
		for _, c := range cats {
			shown = append(shown, em[c]+" "+c)
		}
		fmt.Fprintf(&out, "<p>%s</p>\n\n<p>%s</p>\n\n<p>%s</p>\n", strings.Join(shown, " | "), value(p.ProjectDescription, "No description provided."), formatSources(p.Sources))
		if len(p.Capabilities) > 0 {
			fmt.Fprintf(&out, "<p><strong>Capabilities:</strong> %s</p>\n", strings.Join(p.Capabilities, " · "))
		}
		if len(p.Interfaces) > 0 {
			fmt.Fprintf(&out, "<p><strong>Interfaces:</strong> %s</p>\n", strings.Join(p.Interfaces, " · "))
		}
		out.WriteString("</div>\n\n")
	}
	return strings.TrimSuffix(out.String(), "\n\n")
}
func formatSources(s []projectdata.Source) string {
	v := []string{}
	for _, x := range s {
		v = append(v, fmt.Sprintf("<a href=\"%s\">%s</a>", x.SourceURL, x.Source))
	}
	return strings.Join(v, " | ")
}
func value(p *string, fallback string) string {
	if p == nil || *p == "" {
		return fallback
	}
	return *p
}
func comma(n int) string {
	s := strconv.Itoa(n)
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign = "-"
		s = s[1:]
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return sign + s
}
func signedComma(n int) string {
	if n > 0 {
		return "+" + comma(n)
	}
	return comma(n)
}

var _ = json.Valid
