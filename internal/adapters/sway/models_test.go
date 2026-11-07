package sway

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseWorkspacesFixtures(t *testing.T) {
	for _, v := range []string{"sway-1.8", "sway-1.10"} {
		b, err := os.ReadFile(filepath.Join("testdata", v, "workspaces.json"))
		if err != nil {
			t.Fatal(err)
		}
		ws, err := UnmarshalWorkspaces(b)
		if err != nil {
			t.Fatalf("%s: %v", v, err)
		}
		if len(ws) != 1 {
			t.Fatalf("%s: len %d", v, len(ws))
		}
	}
}
