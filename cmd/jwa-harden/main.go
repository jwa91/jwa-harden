// jwa-harden — wrap a command with `op run --env-file=<nearest .env.template>`.
//
// Scope (ADR 0008 in jwa91/homebrew-tap): one binary, one job. Walks up from
// the current working directory to find the nearest `.env.template`, then
// execs `op run --env-file=<that> -- <command>`. Refuses to run if `op` is
// missing or the user is not signed in to 1Password.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jwa91/jwa-harden/internal/version"
)

const envTemplateName = ".env.template"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "jwa-harden:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stdout)
		return nil
	}
	switch args[0] {
	case "run":
		return cmdRun(args[1:])
	case "doctor":
		return cmdDoctor()
	case "version", "-v", "--version":
		fmt.Println(version.String())
		return nil
	case "-h", "--help", "help":
		printUsage(os.Stdout)
		return nil
	default:
		printUsage(os.Stderr)
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func printUsage(w *os.File) {
	fmt.Fprintln(w, "jwa-harden — wrap a command with op run against the nearest .env.template")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  jwa-harden run -- <command> [args...]   Resolve env via op + exec the command")
	fmt.Fprintln(w, "  jwa-harden doctor                       Check op presence, signin, and template visibility")
	fmt.Fprintln(w, "  jwa-harden version                      Print build info")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Behaviour:")
	fmt.Fprintln(w, "  Walks up from $PWD looking for .env.template, then execs:")
	fmt.Fprintln(w, "    op run --env-file=<found> -- <command>")
	fmt.Fprintln(w, "  The `--` separator is optional but recommended; everything after it is")
	fmt.Fprintln(w, "  passed verbatim to op.")
}

func cmdRun(args []string) error {
	// Allow `jwa-harden run -- cmd args` and `jwa-harden run cmd args`.
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		return errors.New("nothing to run; pass a command after `run`")
	}
	if _, err := exec.LookPath("op"); err != nil {
		return errors.New("`op` (1Password CLI) is not on PATH — install via Homebrew: brew install --cask 1password-cli")
	}
	template, err := nearestEnvTemplate()
	if err != nil {
		return err
	}
	opArgs := append([]string{"run", "--env-file=" + template, "--"}, args...)
	cmd := exec.Command("op", opArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("exec op: %w", err)
	}
	return nil
}

func cmdDoctor() error {
	ok := true
	if _, err := exec.LookPath("op"); err != nil {
		fmt.Println("✗ op not on PATH — install: brew install --cask 1password-cli")
		ok = false
	} else {
		fmt.Println("✓ op installed")
	}
	if ok {
		out, err := exec.Command("op", "whoami").CombinedOutput()
		if err != nil {
			fmt.Println("✗ op not signed in — run: eval $(op signin)")
			fmt.Println("  (or enable the 1Password desktop app's CLI + Touch ID integration)")
			ok = false
		} else {
			fmt.Printf("✓ op signed in: %s", out)
		}
	}
	template, err := nearestEnvTemplate()
	switch {
	case err == nil:
		fmt.Printf("✓ found .env.template: %s\n", template)
	case errors.Is(err, errNoTemplate):
		fmt.Println("! no .env.template found walking up from $PWD — run `jwa-harden run` from inside a project that has one")
	default:
		fmt.Printf("✗ lookup failed: %v\n", err)
		ok = false
	}
	if !ok {
		return errors.New("doctor reported errors above")
	}
	return nil
}

var errNoTemplate = errors.New("no .env.template found walking up from $PWD")

// nearestEnvTemplate walks from $PWD up to / (or the filesystem root on
// non-POSIX, but we target macOS/Linux only) and returns the absolute path
// of the first .env.template it finds. Returns errNoTemplate if none exist.
func nearestEnvTemplate() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	dir := cwd
	for {
		candidate := filepath.Join(dir, envTemplateName)
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("stat %s: %w", candidate, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Defensive: if someone runs from a path with embedded newlines or
	// trims trailing slashes weirdly, surface that explicitly rather than
	// reporting "not found".
	if strings.ContainsRune(cwd, '\n') {
		return "", fmt.Errorf("refusing to walk a path containing a newline: %q", cwd)
	}
	return "", errNoTemplate
}
