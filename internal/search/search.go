package search

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

type Result struct {
	Package     string `json:"package"`
	Name        string `json:"name"`
	Repo        string `json:"repo"` // official | aur
	Version     string `json:"version"`
	Description string `json:"description"`
	URL         string `json:"url,omitempty"`
	Installed   bool   `json:"installed"`
	Popularity  float64 `json:"popularity,omitempty"`
}

// Official busca no sync DB do pacman (local, instantâneo).
func Official(query string) []Result {
	out, err := exec.Command("pacman", "-Ss", query).Output()
	if err != nil {
		return nil
	}
	var results []Result
	lines := strings.Split(string(out), "\n")
	for i := 0; i+1 < len(lines); i += 2 {
		line := strings.TrimSpace(lines[i])
		if line == "" || !strings.Contains(line, "/") {
			continue
		}
		// formato: repo/nome versão [installed]
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		pkgPath := parts[0]
		ver := parts[1]
		installed := strings.Contains(line, "[installed]")
		rp := strings.SplitN(pkgPath, "/", 2)
		if len(rp) != 2 {
			continue
		}
		repo, pkg := rp[0], rp[1]
		desc := ""
		if i+1 < len(lines) {
			desc = strings.TrimSpace(lines[i+1])
		}
		results = append(results, Result{
			Package:     pkg,
			Name:        pkg,
			Repo:        repo,
			Version:     ver,
			Description: desc,
			Installed:   installed,
		})
	}
	return results
}

// AUR busca via RPC v5 (rápido, JSON puro).
func AUR(query string) ([]Result, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	api := fmt.Sprintf("https://aur.archlinux.org/rpc/v5/search/%s?by=name-desc", url.QueryEscape(query))
	resp, err := client.Get(api)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var r struct {
		Results []struct {
			Name        string  `json:"Name"`
			Version     string  `json:"Version"`
			Description string  `json:"Description"`
			URL         string  `json:"URL"`
			Popularity  float64 `json:"Popularity"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	var results []Result
	for _, a := range r.Results {
		results = append(results, Result{
			Package:     a.Name,
			Name:        a.Name,
			Repo:        "aur",
			Version:     a.Version,
			Description: a.Description,
			URL:         a.URL,
			Popularity:  a.Popularity,
		})
	}
	return results, nil
}

// Search combina official + AUR, ordena por relevância (oficial primeiro, depois popularidade).
func Search(query string, limit int) []Result {
	off := Official(query)
	aur, _ := AUR(query)

	// dedup: se existe em official, ignora AUR
	seen := map[string]bool{}
	var all []Result
	for _, o := range off {
		seen[o.Package] = true
		all = append(all, o)
	}
	for _, a := range aur {
		if !seen[a.Package] {
			all = append(all, a)
		}
	}

	// ordena: oficial primeiro, depois por popularidade
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			swap := false
			if all[i].Repo != "official" && all[j].Repo == "official" {
				swap = true
			} else if all[i].Repo == all[j].Repo && all[j].Popularity > all[i].Popularity {
				swap = true
			}
			if swap {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all
}
