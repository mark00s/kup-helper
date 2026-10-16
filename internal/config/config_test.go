package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDotEnv(t *testing.T) {
	in := `
# comment
GITHUB_TOKEN=abc
export KUP_ORG = "my-company"
QUOTED='single'
NOVALUE
=nokey
`
	got := ParseDotEnv(strings.NewReader(in))
	want := map[string]string{"GITHUB_TOKEN": "abc", "KUP_ORG": "my-company", "QUOTED": "single"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func writeEnv(t *testing.T, content string, mode os.FileMode) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(p, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadPrecedence(t *testing.T) {
	local := writeEnv(t, "GITHUB_TOKEN=local\nKUP_ORG=local-org\n", 0o600)
	global := writeEnv(t, "GITHUB_TOKEN=global\nKUP_ORG=global-org\n", 0o600)
	env := map[string]string{"GITHUB_TOKEN": "fromenv"}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }

	cfg := load(lookup, []string{local, global})
	if cfg.Token != "fromenv" {
		t.Errorf("token = %q, want env value", cfg.Token)
	}
	if cfg.Org != "local-org" {
		t.Errorf("org = %q, want first file's value", cfg.Org)
	}
}

func TestLoadMissingFiles(t *testing.T) {
	cfg := load(func(string) (string, bool) { return "", false }, []string{"/nonexistent/.env"})
	if cfg.Token != "" || cfg.Org != "" || len(cfg.Warnings) != 0 {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestLoadWarnsOnOpenPermissions(t *testing.T) {
	p := writeEnv(t, "GITHUB_TOKEN=x\n", 0o644)
	cfg := load(func(string) (string, bool) { return "", false }, []string{p})
	if len(cfg.Warnings) != 1 {
		t.Errorf("warnings = %v, want 1", cfg.Warnings)
	}
}
