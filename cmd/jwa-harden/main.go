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
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jwa91/jwa-harden/internal/version"
)

const envTemplateName = ".env.template"
const usageExitCode = 2

type command struct {
	name    string
	args    string
	summary string
	details []string
	run     func([]string) error
}

func rootCommands() []command {
	return []command{
		{
			name:    "run",
			args:    "[--] <command> [args...]",
			summary: "Resolve env via op and exec the command.",
			details: []string{
				"The `--` separator is optional but recommended; everything after it is passed verbatim to op.",
				"Walks up from $PWD looking for .env.template, then execs:",
				"  op run --env-file=<found> -- <command>",
			},
			run: cmdRun,
		},
		{
			name:    "doctor",
			args:    "[signing]",
			summary: "Check op presence, signin, and template visibility.",
			details: []string{
				"Checks:",
				"  signing  Check macOS signing and notarization prerequisites.",
			},
			run: cmdDoctor,
		},
		{
			name:    "version",
			summary: "Print build info.",
			run:     cmdVersion,
		},
	}
}

type usageError struct {
	message string
}

func (e usageError) Error() string {
	return e.message
}

func (e usageError) ExitCode() int {
	return usageExitCode
}

type commandExitError struct {
	code int
}

func (e commandExitError) Error() string {
	return ""
}

func (e commandExitError) ExitCode() int {
	return e.code
}

func main() {
	os.Exit(mainExit(os.Args[1:]))
}

func mainExit(args []string) int {
	if err := run(args); err != nil {
		if err.Error() == "" {
			return exitCode(err)
		}
		fmt.Fprintln(os.Stderr, "jwa-harden:", err)
		return exitCode(err)
	}
	return 0
}

func run(args []string) error {
	if len(args) == 0 {
		printRootUsage(os.Stdout)
		return nil
	}

	if isHelpArg(args[0]) {
		printRootUsage(os.Stdout)
		return nil
	}
	if args[0] == "help" {
		return cmdHelp(args[1:])
	}
	if args[0] == "-v" || args[0] == "--version" {
		return cmdVersion(nil)
	}
	if strings.HasPrefix(args[0], "-") {
		printRootUsage(os.Stderr)
		return usageErrorf("unknown flag: %s", args[0])
	}
	cmd, ok := commandByName(args[0])
	if !ok {
		printRootUsage(os.Stderr)
		return usageErrorf("unknown command: %s", args[0])
	}
	return cmd.run(args[1:])
}

func cmdHelp(args []string) error {
	if len(args) == 0 || len(args) == 1 && isHelpArg(args[0]) {
		printRootUsage(os.Stdout)
		return nil
	}
	if len(args) > 1 {
		printRootUsage(os.Stderr)
		return usageErrorf("usage: jwa-harden help [command]")
	}
	cmd, ok := commandByName(args[0])
	if !ok {
		printRootUsage(os.Stderr)
		return usageErrorf("unknown command: %s", args[0])
	}
	printCommandUsage(os.Stdout, *cmd)
	return nil
}

func commandByName(name string) (*command, bool) {
	commands := rootCommands()
	for i := range commands {
		if commands[i].name == name {
			return &commands[i], true
		}
	}
	return nil, false
}

func mustCommand(name string) command {
	cmd, ok := commandByName(name)
	if !ok {
		panic("missing command definition: " + name)
	}
	return *cmd
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help"
}

func usageErrorf(format string, args ...any) error {
	return usageError{message: fmt.Sprintf(format, args...)}
}

type exitCoder interface {
	ExitCode() int
}

func exitCode(err error) int {
	var withCode exitCoder
	if errors.As(err, &withCode) {
		return withCode.ExitCode()
	}
	return 1
}

func printRootUsage(w io.Writer) {
	fmt.Fprintln(w, "jwa-harden — wrap a command with op run against the nearest .env.template")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage: jwa-harden <command> [args]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	for _, cmd := range rootCommands() {
		fmt.Fprintf(w, "  %-8s %s\n", cmd.name, cmd.summary)
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Run `jwa-harden help <command>` for command details.")
}

func printCommandUsage(w io.Writer, cmd command) {
	fmt.Fprintf(w, "Usage: jwa-harden %s", cmd.name)
	if cmd.args != "" {
		fmt.Fprintf(w, " %s", cmd.args)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, cmd.summary)
	for _, detail := range cmd.details {
		fmt.Fprintln(w, detail)
	}
}

func cmdRun(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCommandUsage(os.Stdout, mustCommand("run"))
		return nil
	}
	// Allow `jwa-harden run -- cmd args` and `jwa-harden run cmd args`.
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 {
		printCommandUsage(os.Stderr, mustCommand("run"))
		return usageErrorf("nothing to run; pass a command after `run`")
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
			return commandExitError{code: exitErr.ExitCode()}
		}
		return fmt.Errorf("exec op: %w", err)
	}
	return nil
}

func cmdDoctor(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCommandUsage(os.Stdout, mustCommand("doctor"))
		return nil
	}
	if len(args) > 0 {
		switch args[0] {
		case "signing":
			if len(args) == 2 && isHelpArg(args[1]) {
				printDoctorSigningUsage(os.Stdout)
				return nil
			}
			if len(args) > 1 {
				printDoctorSigningUsage(os.Stderr)
				return usageErrorf("unknown doctor signing argument: %s", args[1])
			}
			return cmdDoctorSigning()
		default:
			printCommandUsage(os.Stderr, mustCommand("doctor"))
			if strings.HasPrefix(args[0], "-") {
				return usageErrorf("unknown doctor flag: %s", args[0])
			}
			return usageErrorf("unknown doctor check: %s", args[0])
		}
	}

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

func cmdVersion(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCommandUsage(os.Stdout, mustCommand("version"))
		return nil
	}
	if len(args) > 0 {
		printCommandUsage(os.Stderr, mustCommand("version"))
		return usageErrorf("version takes no arguments")
	}
	fmt.Println(version.String())
	return nil
}

func printDoctorSigningUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: jwa-harden doctor signing")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Check macOS signing and notarization prerequisites.")
}

func cmdDoctorSigning() error {
	ok := true

	if _, err := exec.LookPath("codesign"); err != nil {
		fmt.Println("✗ codesign not on PATH — install Xcode Command Line Tools: xcode-select --install")
		ok = false
	} else {
		fmt.Println("✓ codesign installed")
	}

	if _, err := exec.LookPath("xcrun"); err != nil {
		fmt.Println("✗ xcrun not on PATH — install Xcode Command Line Tools: xcode-select --install")
		ok = false
	} else if out, err := exec.Command("xcrun", "notarytool", "--help").CombinedOutput(); err != nil {
		fmt.Println("✗ xcrun notarytool unavailable")
		if len(out) > 0 {
			fmt.Printf("  %s", out)
			if !strings.HasSuffix(string(out), "\n") {
				fmt.Println()
			}
		}
		ok = false
	} else {
		fmt.Println("✓ xcrun notarytool available")
	}

	if os.Getenv("MACOS_SIGN_IDENTITY") == "" {
		fmt.Println("✗ MACOS_SIGN_IDENTITY not set — add an op:// reference to the repo's .env.template")
		ok = false
	} else {
		fmt.Println("✓ MACOS_SIGN_IDENTITY set")
	}

	if _, err := exec.LookPath("xcrun"); err == nil {
		if out, err := exec.Command("xcrun", "notarytool", "history", "--keychain-profile", "notarytool").CombinedOutput(); err != nil {
			fmt.Println("✗ notarytool keychain profile 'notarytool' unavailable")
			fmt.Println("  create it with: xcrun notarytool store-credentials \"notarytool\" --apple-id <email> --team-id <team> --password <app-specific-password>")
			if len(out) > 0 {
				fmt.Printf("  %s", out)
				if !strings.HasSuffix(string(out), "\n") {
					fmt.Println()
				}
			}
			ok = false
		} else {
			fmt.Println("✓ notarytool keychain profile 'notarytool' works")
		}
	}

	if !ok {
		return errors.New("signing doctor reported errors above")
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
