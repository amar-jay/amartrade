package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root := NewRoot("1.2.3", "today", &stdout, &stderr)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "amartrade 1.2.3 (today)\n" {
		t.Fatalf("unexpected output %q", got)
	}
}

func TestYearsRequiredBeforeNetwork(t *testing.T) {
	root := NewRoot("dev", "unknown", &bytes.Buffer{}, &bytes.Buffer{})
	root.SetArgs([]string{"trademap", "goods", "exports"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "required flag") {
		t.Fatalf("expected required flag error, got %v", err)
	}
}

func TestSearchRequiresQuery(t *testing.T) {
	root := NewRoot("dev", "unknown", &bytes.Buffer{}, &bytes.Buffer{})
	root.SetArgs([]string{"trademap", "search"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Fatalf("expected argument error, got %v", err)
	}
}
