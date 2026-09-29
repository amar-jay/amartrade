package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	command := NewRootCommand("1.2.3", "2026-09-30")
	command.SetArgs([]string{"version"})

	var stdout bytes.Buffer
	if err := execute(command, strings.NewReader(""), &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("execute version command: %v", err)
	}

	want := "amartrade 1.2.3 (2026-09-30)\n"
	if got := stdout.String(); got != want {
		t.Fatalf("unexpected output\ngot:  %q\nwant: %q", got, want)
	}
}

func TestRootVersionFlag(t *testing.T) {
	command := NewRootCommand("1.2.3", "2026-09-30")
	command.SetArgs([]string{"--version"})

	var stdout bytes.Buffer
	if err := execute(command, strings.NewReader(""), &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("execute version flag: %v", err)
	}

	if got := stdout.String(); got != "amartrade 1.2.3\n" {
		t.Fatalf("unexpected output: %q", got)
	}
}
