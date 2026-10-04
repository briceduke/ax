package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/briceduke/ax/internal/checks"
	"github.com/briceduke/ax/internal/compile"
	"github.com/briceduke/ax/internal/logbook"
	"github.com/briceduke/ax/internal/project"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%s", usage())
	}
	switch args[0] {
	case "log":
		return runLog(args[1:])
	case "check":
		return runCheck(args[1:])
	case "compile":
		return runCompile(args[1:])
	default:
		return fmt.Errorf("unknown command %q\n%s", args[0], usage())
	}
}

func usage() string {
	return strings.TrimSpace(`
ax is a harness for agent-driven product development.

Usage:
  ax log friction <one-line>
  ax log observation <finding> --method <how>
  ax log decision <title> [--supersedes <id>]
  ax check [--tier fast|full|slow] [--all]
  ax compile [--target <name>] [--without <scaffold-id>]
`) + "\n"
}

func loadRoot() (string, *project.Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", nil, err
	}
	root, err := project.Find(cwd)
	if err != nil {
		return "", nil, err
	}
	cfg, err := project.Load(root)
	if err != nil {
		return "", nil, err
	}
	project.WarnVersion(cfg, os.Stderr)
	return root, cfg, nil
}

func runLog(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ax log friction|observation|decision ...")
	}
	root, _, err := loadRoot()
	if err != nil {
		return err
	}
	kind := args[0]
	pos, flags, err := partition(args[1:])
	if err != nil {
		return err
	}
	now := time.Now()
	var path string
	switch kind {
	case "friction":
		path, err = logbook.WriteFriction(root, strings.Join(pos, " "), now)
	case "observation":
		path, err = logbook.WriteObservation(root, strings.Join(pos, " "), flags["method"], now)
	case "decision":
		body, errRead := maybeStdin(os.Stdin)
		if errRead != nil {
			return errRead
		}
		path, err = logbook.WriteDecision(root, strings.Join(pos, " "), flags["supersedes"], body, now)
	default:
		return fmt.Errorf("unknown log kind %q", kind)
	}
	if err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

func runCheck(args []string) error {
	pos, flags, err := partition(args)
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return fmt.Errorf("unexpected argument %q", pos[0])
	}
	tier := flags["tier"]
	if tier == "" {
		tier = checks.TierFull
	}
	_, all := flags["all"]
	root, _, err := loadRoot()
	if err != nil {
		return err
	}
	return checks.Run(root, checks.Options{Tier: tier, All: all}, os.Stdout)
}

func runCompile(args []string) error {
	pos, flags, err := partition(args)
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return fmt.Errorf("unexpected argument %q", pos[0])
	}
	root, _, err := loadRoot()
	if err != nil {
		return err
	}
	return compile.Run(root, compile.Options{Target: flags["target"], Without: flags["without"]}, os.Stdout)
}

func partition(args []string) ([]string, map[string]string, error) {
	flags := map[string]string{}
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			pos = append(pos, a)
			continue
		}
		name := strings.TrimPrefix(a, "--")
		if name == "all" && !strings.Contains(name, "=") {
			flags["all"] = "true"
			continue
		}
		value := ""
		if n, v, ok := strings.Cut(name, "="); ok {
			name, value = n, v
		} else {
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag --%s needs a value", name)
			}
			i++
			value = args[i]
		}
		flags[name] = value
	}
	return pos, flags, nil
}

func maybeStdin(in *os.File) (string, error) {
	stat, err := in.Stat()
	if err != nil {
		return "", err
	}
	if stat.Mode()&os.ModeCharDevice != 0 {
		return "", nil
	}
	data, err := io.ReadAll(in)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
