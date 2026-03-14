package main

import (
	"os"
	"strings"
	"testing"
)

func TestRun_Version(t *testing.T) {
	stdout, err := os.CreateTemp("", "stdout-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(stdout.Name())
	defer stdout.Close()

	stderr, err := os.CreateTemp("", "stderr-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(stderr.Name())
	defer stderr.Close()

	code := run([]string{"--version"}, stdout, stderr, strings.NewReader(""))
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}

	stdout.Seek(0, 0)
	buf := make([]byte, 1024)
	n, _ := stdout.Read(buf)
	output := string(buf[:n])

	if !strings.Contains(output, "gf-claude-quota version 0.1.0") {
		t.Errorf("output = %q, want to contain version string", output)
	}
}

func TestRun_InvalidFlag(t *testing.T) {
	stdout, err := os.CreateTemp("", "stdout-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(stdout.Name())
	defer stdout.Close()

	stderr, err := os.CreateTemp("", "stderr-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(stderr.Name())
	defer stderr.Close()

	code := run([]string{"--invalid-flag"}, stdout, stderr, strings.NewReader(""))
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestRun_InvalidColorMode(t *testing.T) {
	stdout, err := os.CreateTemp("", "stdout-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(stdout.Name())
	defer stdout.Close()

	stderr, err := os.CreateTemp("", "stderr-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(stderr.Name())
	defer stderr.Close()

	code := run([]string{"--color=invalid"}, stdout, stderr, strings.NewReader(""))
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}

	stderr.Seek(0, 0)
	buf := make([]byte, 1024)
	n, _ := stderr.Read(buf)
	errOutput := string(buf[:n])
	if !strings.Contains(errOutput, "invalid color mode") {
		t.Errorf("stderr = %q, want to contain 'invalid color mode'", errOutput)
	}
}

func TestRun_MutuallyExclusiveFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"json+oneline", []string{"--json", "--oneline"}},
		{"json+statusline", []string{"--json", "--statusline"}},
		{"oneline+statusline", []string{"--oneline", "--statusline"}},
		{"json+format", []string{"--json", "--format={5h}"}},
		{"statusline+format", []string{"--statusline", "--format={5h}"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, err := os.CreateTemp("", "stdout-*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(stdout.Name())
			defer stdout.Close()

			stderr, err := os.CreateTemp("", "stderr-*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(stderr.Name())
			defer stderr.Close()

			code := run(tt.args, stdout, stderr, strings.NewReader(""))
			if code != 2 {
				t.Errorf("exit code = %d, want 2", code)
			}

			stderr.Seek(0, 0)
			buf := make([]byte, 1024)
			n, _ := stderr.Read(buf)
			errOutput := string(buf[:n])
			if !strings.Contains(errOutput, "mutually exclusive") {
				t.Errorf("stderr = %q, want to contain 'mutually exclusive'", errOutput)
			}
		})
	}
}
