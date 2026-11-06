package store

import (
	"io/fs"
	"strings"
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{"001_init.sql", 1, false},
		{"12_add_thing.sql", 12, false},
		{"init.sql", 0, true},
		{"abc_init.sql", 0, true},
		{"0_zero.sql", 0, true},
	}
	for _, tc := range tests {
		got, err := parseVersion(tc.name)
		if (err != nil) != tc.wantErr || got != tc.want {
			t.Errorf("parseVersion(%q) = %d, %v", tc.name, got, err)
		}
	}
}

// Guards against duplicate or badly named migration files being committed.
func TestEmbeddedMigrationsAreValid(t *testing.T) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int]string{}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		v, err := parseVersion(e.Name())
		if err != nil {
			t.Fatal(err)
		}
		if prev, dup := seen[v]; dup {
			t.Fatalf("migrations %s and %s share version %d", prev, e.Name(), v)
		}
		seen[v] = e.Name()
	}
	if len(seen) == 0 {
		t.Fatal("no migrations found")
	}
}
