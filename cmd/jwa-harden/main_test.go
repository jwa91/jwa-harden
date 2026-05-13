package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCLIContract(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "no args prints root usage",
			wantStdout: "Usage: jwa-harden <command> [args]",
		},
		{
			name:       "help prints root usage",
			args:       []string{"help"},
			wantStdout: "Commands:",
		},
		{
			name:       "root help flag prints root usage",
			args:       []string{"-h"},
			wantStdout: "Usage: jwa-harden <command> [args]",
		},
		{
			name:       "run help prints command usage",
			args:       []string{"run", "-h"},
			wantStdout: "Usage: jwa-harden run [--] <command> [args...]",
		},
		{
			name:       "doctor help prints command usage",
			args:       []string{"doctor", "-h"},
			wantStdout: "Usage: jwa-harden doctor [signing]",
		},
		{
			name:       "doctor signing help prints nested usage",
			args:       []string{"doctor", "signing", "-h"},
			wantStdout: "Usage: jwa-harden doctor signing",
		},
		{
			name:       "version help prints command usage",
			args:       []string{"version", "-h"},
			wantStdout: "Usage: jwa-harden version",
		},
		{
			name:       "version prints build info",
			args:       []string{"version"},
			wantStdout: "jwa-harden dev (commit none, built unknown)",
		},
		{
			name:       "unknown command is usage error",
			args:       []string{"wat"},
			wantCode:   usageExitCode,
			wantStderr: "jwa-harden: unknown command: wat",
		},
		{
			name:       "unknown root flag is usage error",
			args:       []string{"--wat"},
			wantCode:   usageExitCode,
			wantStderr: "jwa-harden: unknown flag: --wat",
		},
		{
			name:       "bad lint flag path is still an unknown command usage error",
			args:       []string{"lint", "--repo", "/definitely/missing"},
			wantCode:   usageExitCode,
			wantStderr: "jwa-harden: unknown command: lint",
		},
		{
			name:       "unknown doctor flag is usage error",
			args:       []string{"doctor", "--wat"},
			wantCode:   usageExitCode,
			wantStderr: "jwa-harden: unknown doctor flag: --wat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runCLI(t, tt.args...)
			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", code, tt.wantCode, stdout, stderr)
			}
			if tt.wantStdout != "" && !strings.Contains(stdout, tt.wantStdout) {
				t.Fatalf("stdout missing %q:\n%s", tt.wantStdout, stdout)
			}
			if tt.wantStderr != "" && !strings.Contains(stderr, tt.wantStderr) {
				t.Fatalf("stderr missing %q:\n%s", tt.wantStderr, stderr)
			}
			if tt.wantStderr == "" && stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
		})
	}
}

func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()

	cmdArgs := append([]string{"-test.run=TestMainExitHelper", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Env = append(os.Environ(), "JWA_HARDEN_TEST_MAIN=1")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return 0, stdout.String(), stderr.String()
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), stdout.String(), stderr.String()
	}
	t.Fatalf("running helper: %v", err)
	return 0, "", ""
}

func TestMainExitHelper(t *testing.T) {
	if os.Getenv("JWA_HARDEN_TEST_MAIN") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Exit(mainExit(os.Args[i+1:]))
		}
	}
	os.Exit(usageExitCode)
}
