package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

// command is one CLI subcommand.
type command struct {
	name    string
	summary string
	run     func(args []string, out, errOut io.Writer) int
}

var commands []command

func init() {
	commands = []command{
		{"info", "show NTFS volume geometry", runInfo},
		{"list", "list deleted files on a volume", runList},
		{"recover", "recover a deleted file from a volume", runRecover},
	}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run dispatches to a subcommand and returns the process exit code.
func run(args []string, out, errOut io.Writer) int {
	if len(args) == 0 {
		usage(errOut)
		return exitUsage
	}

	switch args[0] {
	case "-h", "-help", "--help", "help":
		usage(out)
		return exitOK
	case "-version", "--version", "version":
		fmt.Fprintf(out, "undelete %s\n", version)
		return exitOK
	}

	for _, c := range commands {
		if c.name == args[0] {
			return c.run(args[1:], out, errOut)
		}
	}

	fmt.Fprintf(errOut, "undelete: unknown command %q\n\n", args[0])
	usage(errOut)
	return exitUsage
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "undelete - recover deleted files from an NTFS volume")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  undelete <command> [options]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, c := range commands {
		fmt.Fprintf(w, "  %-8s %s\n", c.name, c.summary)
	}
	fmt.Fprintln(w, "  version  print version")
	fmt.Fprintln(w, "  help     show this help")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'undelete <command> -h' for command options.")
}

// newFlagSet builds a FlagSet that writes to errOut and returns errors
// instead of exiting, so run() controls the exit code.
func newFlagSet(name, usageLine string, errOut io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(errOut)
	fs.Usage = func() {
		fmt.Fprintf(errOut, "Usage: undelete %s\n\nOptions:\n", usageLine)
		fs.PrintDefaults()
	}
	return fs
}

// parseFlags parses args and reports whether the caller should continue.
// The second return value is the exit code to use when it should not.
func parseFlags(fs *flag.FlagSet, args []string) (ok bool, code int) {
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return false, exitOK
		}
		return false, exitUsage
	}
	return true, exitOK
}

func runInfo(args []string, out, errOut io.Writer) int {
	fs := newFlagSet("info", "info <volume>", errOut)
	if ok, code := parseFlags(fs, args); !ok {
		return code
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return exitUsage
	}
	volume := fs.Arg(0)

	fmt.Fprintf(out, "info: not implemented yet (volume=%s)\n", volume)
	return exitOK
}

func runList(args []string, out, errOut io.Writer) int {
	fs := newFlagSet("list", "list [options] <volume>", errOut)
	name := fs.String("name", "", "only show files whose name matches this glob")
	minSize := fs.Int64("min-size", 0, "only show files at least this many bytes")
	format := fs.String("format", "table", "output format: table or json")
	if ok, code := parseFlags(fs, args); !ok {
		return code
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return exitUsage
	}
	if *format != "table" && *format != "json" {
		fmt.Fprintf(errOut, "undelete list: invalid -format %q (want table or json)\n", *format)
		return exitUsage
	}
	volume := fs.Arg(0)

	fmt.Fprintf(out, "list: not implemented yet (volume=%s name=%q min-size=%d format=%s)\n",
		volume, *name, *minSize, *format)
	return exitOK
}

func runRecover(args []string, out, errOut io.Writer) int {
	fs := newFlagSet("recover", "recover [options] <volume> <record#>", errOut)
	outDir := fs.String("o", "", "output directory (required, should be on a different volume)")
	force := fs.Bool("force", false, "overwrite existing output files")
	if ok, code := parseFlags(fs, args); !ok {
		return code
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return exitUsage
	}
	if *outDir == "" {
		fmt.Fprintln(errOut, "undelete recover: -o <outdir> is required")
		return exitUsage
	}
	volume, record := fs.Arg(0), fs.Arg(1)

	// Fail before doing any work if the destination is not the sanctioned
	// drive. Real writes later must still go through safeCreate/safeMkdirAll.
	if err := checkWriteTarget(*outDir); err != nil {
		fmt.Fprintf(errOut, "undelete recover: %v\n", err)
		return exitError
	}

	fmt.Fprintf(out, "recover: not implemented yet (volume=%s record=%s out=%s force=%t)\n",
		volume, record, *outDir, *force)
	return exitOK
}
