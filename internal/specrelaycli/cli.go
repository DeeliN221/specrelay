// Package specrelaycli implements the standalone specrelay CLI entrypoint and command dispatch.
package specrelaycli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/DeeliN221/specrelay/internal/app"
)

func Main(args []string, stdout, stderr io.Writer) int {
	// Main maps CLI arguments to process exit codes for shell and agent use.
	return (&cli{out: stdout, err: stderr, now: time.Now}).run(args)
}

func (c *cli) run(args []string) int {
	// run dispatches the small non-interactive command surface.
	if c.now == nil {
		c.now = time.Now
	}
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		c.printHelp()
		return 0
	}
	if args[0] == "--version" || args[0] == "-v" {
		fmt.Fprintln(c.out, resolvedVersion())
		return 0
	}
	var err error
	switch args[0] {
	case "list", "l":
		err = c.listCmd(args[1:])
	case "install", "i":
		err = c.installCmd(args[1:])
	case "create":
		err = c.createCmd(args[1:])
	case "status":
		err = c.statusCmd(args[1:])
	case "validate":
		err = c.validateCmd(args[1:])
	case "archive":
		err = c.archiveCmd(args[1:])
	case "flow":
		err = app.Run(args[1:], os.Stdin, c.out, c.err)
	default:
		err = fmt.Errorf("unknown command: %s", args[0])
	}
	if err != nil {
		fmt.Fprintln(c.err, err)
		return 1
	}
	return 0
}

func (c *cli) printHelp() {
	// printHelp separates user commands from automation interfaces.
	fmt.Fprintln(c.out, "specrelay "+resolvedVersion())
	fmt.Fprintln(c.out, "")
	fmt.Fprintln(c.out, "Daily commands:")
	fmt.Fprintln(c.out, "  list | l [--json]")
	fmt.Fprintln(c.out, "  install | i [--global | -g]")
	fmt.Fprintln(c.out, "  flow <status|watch|run|config|clean|...>")
	fmt.Fprintln(c.out, "")
	fmt.Fprintln(c.out, "Automation interface:")
	fmt.Fprintln(c.out, "  create")
	fmt.Fprintln(c.out, "  status <change> [--json]")
	fmt.Fprintln(c.out, "  validate <change> [--json]")
	fmt.Fprintln(c.out, "  archive <change> --yes")
}

func resolvedVersion() string {
	// resolvedVersion reports the release tag when specrelay was installed or run from the source repository.
	if version != "" && version != "dev" {
		return version
	}
	// Installed module metadata is authoritative; its cache may sit below an unrelated Git repository.
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	if tag, err := sourceGitTag(); err == nil && tag != "" {
		return tag
	}
	return version
}

func sourceGitTag() (string, error) {
	// sourceGitTag reads git describe only when the current repository is specrelay itself.
	root, err := sourceRoot()
	if err != nil {
		return "", err
	}
	top, err := exec.Command("git", "-C", root, "rev-parse", "--show-toplevel").Output()
	if err != nil || filepath.Clean(strings.TrimSpace(string(top))) != filepath.Clean(root) {
		return "", errors.New("module source is not its own Git checkout")
	}

	describeCmd := exec.Command("git", "-C", root, "describe", "--tags", "--always", "--dirty")
	describeOut, err := describeCmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(describeOut)), nil
}

func sourceRoot() (string, error) {
	// sourceRoot locates the specrelay checkout from compiled source paths before consulting the current directory.
	if _, file, _, ok := runtime.Caller(0); ok {
		if root, err := specrelayModuleRoot(filepath.Dir(file)); err == nil {
			return root, nil
		}
	}
	rootCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	rootOut, err := rootCmd.Output()
	if err != nil {
		return "", err
	}
	return specrelayModuleRoot(strings.TrimSpace(string(rootOut)))
}

func specrelayModuleRoot(start string) (string, error) {
	// specrelayModuleRoot walks upward until it finds the specrelay module root.
	for dir := start; ; dir = filepath.Dir(dir) {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && strings.Contains(string(data), "module github.com/DeeliN221/specrelay\n") {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("specrelay module root not found")
		}
	}
}
