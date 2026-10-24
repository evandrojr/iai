package aur

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Info struct {
	Name           string `json:"Name"`
	Version        string `json:"Version"`
	Description    string `json:"Description"`
	URL            string `json:"URL"`
	URLPath        string `json:"URLPath"`
	Maintainer     string `json:"Maintainer"`
	NumVotes       int    `json:"NumVotes"`
	Popularity     float64 `json:"Popularity"`
	OutOfDate      interface{} `json:"OutOfDate"`
	OutOfDateBool  bool
	FirstSubmitted int64 `json:"FirstSubmitted"`
}

type rpcResponse struct {
	Results []Info `json:"results"`
}

func Check(pkg string) (*Info, error) {
	url := fmt.Sprintf("https://aur.archlinux.org/rpc/v5/info?arg[]=%s", pkg)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("consultando AUR: %w", err)
	}
	defer resp.Body.Close()
	var r rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("decodando AUR: %w", err)
	}
	if len(r.Results) == 0 {
		return nil, nil
	}
	info := &r.Results[0]
	switch v := info.OutOfDate.(type) {
	case bool:
		info.OutOfDateBool = v
	case float64:
		info.OutOfDateBool = v != 0
	case int:
		info.OutOfDateBool = v != 0
	}
	return info, nil
}

func FetchPKGBUILD(pkg string) (string, error) {
	url := fmt.Sprintf("https://aur.archlinux.org/cgit/aur.git/plain/PKGBUILD?h=%s", pkg)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("baixando PKGBUILD: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return "", fmt.Errorf("PKGBUILD não encontrado para %s", pkg)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func IsOfficial(pkg string) (repo string, found bool) {
	// Implementação via pacman -Si é feita em install package para evitar
	// dependência externa neste pacote puro.
	return "", false
}

func ExtractSources(pkgbuild string) []string {
	var urls []string
	for _, line := range strings.Split(pkgbuild, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "source=") && !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "\t") {
			continue
		}
		if strings.HasPrefix(line, "source=") {
			line = strings.TrimPrefix(line, "source=")
		}
		// source=("url" "url2")
		line = strings.TrimPrefix(line, "(")
		line = strings.TrimSuffix(line, ")")
		for _, part := range strings.Fields(line) {
			part = strings.Trim(part, `"'`)
			if strings.HasPrefix(part, "http") || strings.HasPrefix(part, "ftp") {
				urls = append(urls, part)
			}
		}
	}
	return urls
}
