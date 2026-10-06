package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDevModeSources covers both ways a run declares itself development, and
// the explicit off switch that has to beat the marker file for a learner who
// studies from a checkout.
func TestDevModeSources(t *testing.T) {
	for _, tc := range []struct {
		name   string
		env    string
		marker bool
		want   bool
	}{
		{"nothing set", "", false, false},
		{"env on", "1", false, true},
		{"env true", "TRUE", false, true},
		{"env off", "0", false, false},
		{"marker alone", "", true, true},
		{"env off beats the marker", "false", true, false},
		{"env on without a marker", "yes", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			t.Setenv("DOJO_DEV", tc.env)
			if tc.marker {
				if err := os.WriteFile(filepath.Join(dir, DevMarker), nil, 0o644); err != nil {
					t.Fatalf("write marker: %v", err)
				}
			}
			if got := DevMode(); got != tc.want {
				t.Errorf("DevMode = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestValidateID(t *testing.T) {
	for _, id := range []string{"standard", "ha", "upgrade-1.34", "cp1", "my_profile"} {
		if err := ValidateID(id); err != nil {
			t.Errorf("ValidateID(%q): %v", id, err)
		}
	}
	for _, id := range []string{"", ".", "..", "../outside", "a/../../outside", "/tmp/outside", `a\b`, "-standard", "two words", "a\nb", "$(whoami)", "Uppercase", "café"} {
		if err := ValidateID(id); err == nil {
			t.Errorf("ValidateID(%q) accepted an unsafe identifier", id)
		}
	}
}

func TestEnvironmentDirectoryLifecycle(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("DOJO_HOME", home)
	want := filepath.Join(home, "env", "upgrade-1.34")
	if got, err := EnvDirPath("upgrade-1.34"); err != nil || got != want {
		t.Fatalf("EnvDirPath = %q, %v; want %q", got, err, want)
	}
	if err := RemoveEnvDir("upgrade-1.34"); err != nil {
		t.Fatalf("remove absent environment: %v", err)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("checking or deleting an absent environment created its home: %v", err)
	}
	dir, err := EnvDir("upgrade-1.34")
	if err != nil || dir != want {
		t.Fatalf("EnvDir = %q, %v; want %q", dir, err, want)
	}
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "keep")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Cleanup may remove a nested symlink, but must never walk its target.
	if err := os.Symlink(outside, filepath.Join(dir, "external")); err != nil {
		t.Fatal(err)
	}
	other, err := EnvDir("other")
	if err != nil {
		t.Fatal(err)
	}
	if err := RemoveEnvDir("upgrade-1.34"); err != nil {
		t.Fatalf("remove environment: %v", err)
	}
	for _, kept := range []string{sentinel, other} {
		if _, err := os.Stat(kept); err != nil {
			t.Errorf("cleanup removed unrelated path %q: %v", kept, err)
		}
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("environment directory still exists: %v", err)
	}
}

func TestEnvironmentDirectoriesRejectTraversal(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	t.Setenv("DOJO_HOME", home)
	sentinel := filepath.Join(base, "outside", "keep")
	if err := os.MkdirAll(filepath.Dir(sentinel), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sentinel, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"../../outside", "..", ".", sentinel} {
		if _, err := EnvDir(id); err == nil {
			t.Errorf("EnvDir(%q) accepted traversal", id)
		}
		if _, err := EnvDirPath(id); err == nil {
			t.Errorf("EnvDirPath(%q) accepted traversal", id)
		}
		if err := RemoveEnvDir(id); err == nil {
			t.Errorf("RemoveEnvDir(%q) accepted traversal", id)
		}
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("unsafe profile touched an unrelated file: %v", err)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("rejected profiles created state directories: %v", err)
	}
}

func TestEnvironmentDirectoriesRejectSymlinks(t *testing.T) {
	for _, component := range []string{"env", "profile"} {
		t.Run(component, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("DOJO_HOME", home)
			outside := t.TempDir()
			sentinel := filepath.Join(outside, "keep")
			if err := os.WriteFile(sentinel, []byte("keep"), 0o600); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(home, "env")
			if component == "profile" {
				if err := os.Mkdir(link, 0o700); err != nil {
					t.Fatal(err)
				}
				link = filepath.Join(link, "standard")
			}
			if err := os.Symlink(outside, link); err != nil {
				t.Fatal(err)
			}
			if _, err := EnvDir("standard"); err == nil {
				t.Error("EnvDir followed a symlink")
			}
			if _, err := EnvDirPath("standard"); err == nil {
				t.Error("EnvDirPath accepted a symlink")
			}
			if err := RemoveEnvDir("standard"); err == nil {
				t.Error("RemoveEnvDir accepted a symlink")
			}
			if _, err := os.Stat(sentinel); err != nil {
				t.Fatalf("symlink target was touched: %v", err)
			}
			if _, err := os.Stat(filepath.Join(outside, "standard")); !os.IsNotExist(err) {
				t.Fatalf("EnvDir created a directory through a symlink: %v", err)
			}
		})
	}
}

func TestLoadConfigRejectsUnsafeSelections(t *testing.T) {
	for _, field := range []string{"profile", "curriculum"} {
		t.Run(field, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("DOJO_HOME", home)
			if err := os.WriteFile(filepath.Join(home, "config.yaml"), []byte(field+": ../../outside\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(); err == nil {
				t.Fatalf("Load accepted unsafe %s", field)
			}
		})
	}
}
