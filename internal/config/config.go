// Package config resolves settings from environment variables and .env files.
//
// Precedence (highest first): real environment variables, .env in the current
// directory, .env in the user config directory. CLI flags are applied by the
// caller on top of the values returned here.
package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the settings that rarely change between runs.
type Config struct {
	Token string // GITHUB_TOKEN
	Orgs  string // KUP_ORGS (comma-separated)

	// Warnings are non-fatal problems worth showing to the user.
	Warnings []string
}

// Load resolves the configuration using the process environment and the
// default .env locations.
func Load() Config {
	paths := []string{".env"}
	if dir, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(dir, "kup-helper", ".env"))
	}
	return load(os.LookupEnv, paths)
}

func load(lookup func(string) (string, bool), paths []string) Config {
	var cfg Config
	var files []map[string]string

	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		if info, err := f.Stat(); err == nil && info.Mode().Perm()&0o077 != 0 {
			cfg.Warnings = append(cfg.Warnings,
				fmt.Sprintf("%s is readable by other users; run: chmod 600 %s", p, p))
		}
		files = append(files, ParseDotEnv(f))
		f.Close()
	}

	get := func(key string) string {
		if v, ok := lookup(key); ok && v != "" {
			return v
		}
		for _, m := range files {
			if v := m[key]; v != "" {
				return v
			}
		}
		return ""
	}

	cfg.Token = get("GITHUB_TOKEN")
	cfg.Orgs = get("KUP_ORGS")
	return cfg
}

// ParseDotEnv reads KEY=VALUE lines. Blank lines and lines starting with '#'
// are ignored, an optional "export " prefix is accepted, and values may be
// wrapped in single or double quotes.
func ParseDotEnv(r io.Reader) map[string]string {
	out := make(map[string]string)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = unquote(strings.TrimSpace(val))
	}
	return out
}

func unquote(v string) string {
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		return v[1 : len(v)-1]
	}
	return v
}
