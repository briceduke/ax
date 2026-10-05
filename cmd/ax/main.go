package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/briceduke/ax/internal/ablate"
	"github.com/briceduke/ax/internal/agent"
	"github.com/briceduke/ax/internal/checks"
	"github.com/briceduke/ax/internal/compile"
	"github.com/briceduke/ax/internal/eval"
	"github.com/briceduke/ax/internal/install"
	"github.com/briceduke/ax/internal/logbook"
	"github.com/briceduke/ax/internal/onboard"
	"github.com/briceduke/ax/internal/pack"
	"github.com/briceduke/ax/internal/project"
	"github.com/briceduke/ax/internal/retro"
	"github.com/briceduke/ax/internal/upgrade"
	"github.com/briceduke/ax/internal/upstream"
	"github.com/briceduke/ax/internal/version"
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
	case "eval":
		return runEval(args[1:])
	case "retro":
		return runRetro(args[1:])
	case "ablate":
		return runAblate(args[1:])
	case "propose-upstream":
		return runProposeUpstream(args[1:])
	case "upgrade":
		return runUpgrade(args[1:])
	case "pack":
		return runPack(args[1:])
	case "init":
		return runInit(args[1:])
	case "adopt":
		return runAdopt(args[1:])
	case "notes":
		return runNotes(args[1:])
	case "install":
		return runInstall(args[1:])
	case "doctor":
		return runDoctor(args[1:])
	case "version":
		return runVersion()
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
  ax eval [--without <scaffold-id>] [--runs <n>]
  ax retro
  ax ablate
  ax propose-upstream [--friction <id>] [--layer machinery|schema|target|pack] [--change <text>] [--helps-others] [--submit]
  ax upgrade [--from <dir>] [--submit] [--skip-binary]
  ax pack add <dir>
  ax pack extract <dir> --id <id> [--file <path>] [--replace old=PARAM]
  ax pack bootstrap <discipline>
	ax init [--name <name>] [--intent <text>] [--targets cursor,claude-code] [--notes record] [--interview]
  ax adopt
  ax notes show|hide
  ax install [--home <dir>]
  ax doctor [--require-container]
  ax version
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
	if err := compile.Refresh(root); err != nil {
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
	opts := compile.Options{Target: flags["target"], Without: flags["without"]}
	if os.Getenv("AX_COMPILE_AGENT") == "1" {
		if r := agent.Detect(); r != nil {
			opts.UseAgent = true
			opts.Agent = r
		} else {
			fmt.Fprintln(os.Stdout, "compile agent unavailable; using assembler")
		}
	}
	return compile.Run(root, opts, os.Stdout)
}

func runEval(args []string) error {
	fs := newFlagSet("eval")
	without := fs.String("without", "", "compile without this scaffold first")
	runs := fs.Int("runs", 0, "override eval run count")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, _, err := loadRoot()
	if err != nil {
		return err
	}
	orig := root
	if *without != "" {
		if err := compile.Run(root, compile.Options{Without: *without}, os.Stdout); err != nil {
			return err
		}
		root = compile.WorktreeDir(root, *without)
	}
	_, err = eval.Run(root, eval.Options{
		Without:  *without,
		Runs:     *runs,
		RunCheck: checks.RunNamed,
		LogRoot:  orig,
		Runner:   agent.Detect(),
	}, os.Stdout)
	if err != nil {
		return err
	}
	return compile.Refresh(orig)
}

func runRetro(args []string) error {
	fs := newFlagSet("retro")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, _, err := loadRoot()
	if err != nil {
		return err
	}
	_, err = retro.Run(root, retro.Options{}, os.Stdout)
	return err
}

func runAblate(args []string) error {
	fs := newFlagSet("ablate")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, _, err := loadRoot()
	if err != nil {
		return err
	}
	return ablate.Run(root, ablate.Options{RunCheck: checks.RunNamed}, os.Stdout)
}

func runProposeUpstream(args []string) error {
	fs := newFlagSet("propose-upstream")
	friction := fs.String("friction", "", "friction id")
	layer := fs.String("layer", "machinery", "machinery, schema, target, or pack")
	change := fs.String("change", "", "proposed change")
	helps := fs.Bool("helps-others", false, "would this help another project")
	submit := fs.Bool("submit", false, "create a GitHub issue with gh (does not merge)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, _, err := loadRoot()
	if err != nil {
		return err
	}
	opts := upstream.Options{
		Friction:    *friction,
		Layer:       *layer,
		Change:      *change,
		HelpsOthers: *helps,
		Submit:      *submit,
	}
	if *submit {
		if _, err := exec.LookPath("gh"); err != nil {
			return fmt.Errorf("--submit requires gh on PATH")
		}
		opts.Command = ghCommand
	}
	return upstream.Run(root, opts, os.Stdout)
}

func runUpgrade(args []string) error {
	fs := newFlagSet("upgrade")
	from := fs.String("from", "", "local machinery directory with a VERSION file")
	submit := fs.Bool("submit", false, "file a GitHub issue with gh (does not merge)")
	skipBinary := fs.Bool("skip-binary", false, "do not replace this ax executable")
	if err := fs.Parse(args); err != nil {
		return err
	}
	opts := upgrade.Options{From: *from, Submit: *submit, SkipBinary: *skipBinary}
	if !*skipBinary {
		opts.Get = upgrade.DefaultGet
	}
	if *from == "" {
		if _, err := exec.LookPath("git"); err == nil {
			opts.Fetcher = upgrade.GitFetch("", agent.SystemRun)
		}
	}
	if *submit {
		if _, err := exec.LookPath("gh"); err != nil {
			return fmt.Errorf("--submit requires gh on PATH")
		}
		opts.Command = ghCommand
	}
	root, _, err := loadRoot()
	if err != nil {
		if *skipBinary {
			return err
		}
		return upgrade.Self(opts, os.Stdout)
	}
	if *from == "" && opts.Fetcher == nil {
		return fmt.Errorf("upgrade needs --from <dir> (git not on PATH)")
	}
	return upgrade.Run(root, opts, os.Stdout)
}

func runPack(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ax pack add|extract|bootstrap ...")
	}
	switch args[0] {
	case "add":
		if len(args) != 2 {
			return fmt.Errorf("usage: ax pack add <dir>")
		}
		root, _, err := loadRoot()
		if err != nil {
			return err
		}
		return pack.Add(root, args[1], os.Stdout)
	case "extract":
		fs := newFlagSet("pack extract")
		id := fs.String("id", "", "pack id")
		var files stringList
		var replacements stringMap
		fs.Var(&files, "file", "project file to include (repeatable)")
		fs.Var(&replacements, "replace", "old=PARAM (repeatable)")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return fmt.Errorf("usage: ax pack extract <dir> --id <id> [--file path] [--replace old=PARAM]")
		}
		root, _, err := loadRoot()
		if err != nil {
			return err
		}
		return pack.Extract(root, fs.Arg(0), *id, files, replacements, os.Stdout)
	case "bootstrap":
		if len(args) != 2 {
			return fmt.Errorf("usage: ax pack bootstrap <discipline>")
		}
		root, _, err := loadRoot()
		if err != nil {
			return err
		}
		return pack.Bootstrap(root, args[1], os.Stdout)
	default:
		return fmt.Errorf("unknown pack command %q", args[0])
	}
}

func runInit(args []string) error {
	fs := newFlagSet("init")
	name := fs.String("name", "", "project name")
	intent := fs.String("intent", "", "what is being built")
	targets := fs.String("targets", "cursor,claude-code", "comma-separated editor targets")
	notes := fs.String("notes", "", "record if the notes are the work")
	interview := fs.Bool("interview", false, "ask the full start questions")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return onboard.Init(cwd, onboard.InitOptions{
		Name:      *name,
		Intent:    *intent,
		Targets:   *targets,
		Notes:     *notes,
		Interview: *interview,
		IsTTY:     isTTY(os.Stdin),
		Stdin:     os.Stdin,
	}, os.Stdout)
}

func runAdopt(args []string) error {
	fs := newFlagSet("adopt")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return onboard.Adopt(cwd, time.Now(), os.Stdout)
}

func runNotes(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: ax notes show|hide")
	}
	root, _, err := loadRoot()
	if err != nil {
		return err
	}
	switch args[0] {
	case "show":
		if err := project.ShowNotes(root); err != nil {
			return err
		}
	case "hide":
		if err := project.HideNotes(root); err != nil {
			return err
		}
	default:
		return fmt.Errorf("usage: ax notes show|hide")
	}
	return compile.Refresh(root)
}

func runInstall(args []string) error {
	fs := newFlagSet("install")
	home := fs.String("home", "", "directory that contains .cursor and .claude")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir := *home
	if dir == "" {
		var err error
		dir, err = os.UserHomeDir()
		if err != nil {
			return err
		}
	}
	if err := install.WriteSkills(dir); err != nil {
		return err
	}
	fmt.Println("wrote /ax skills")
	return nil
}

func runVersion() error {
	fmt.Println(version.Version)
	return nil
}

func runDoctor(args []string) error {
	fs := newFlagSet("doctor")
	requireContainer := fs.Bool("require-container", false, "fail if Docker cannot build and run ax check")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, _, err := loadRoot()
	if err != nil {
		return err
	}
	return onboard.Doctor(root, onboard.DoctorOptions{RequireContainer: *requireContainer}, os.Stdout)
}

func ghCommand(args ...string) (string, error) {
	return agent.SystemRun("", "gh", args...)
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func isTTY(in *os.File) bool {
	stat, err := in.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

type stringMap map[string]string

func (m *stringMap) String() string { return fmt.Sprint(*m) }
func (m *stringMap) Set(v string) error {
	old, param, ok := strings.Cut(v, "=")
	if !ok || old == "" || param == "" {
		return fmt.Errorf("replace value must be old=PARAM")
	}
	if *m == nil {
		*m = map[string]string{}
	}
	(*m)[old] = param
	return nil
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
