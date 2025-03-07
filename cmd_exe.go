// Copyright 2015 The go-python Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-python/gopy/bind"
	"github.com/google/subcommands"
)

// python packaging links:
// https://pypi.org/
// https://packaging.python.org/tutorials/packaging-projects/
// https://docs.python.org/3/tutorial/modules.html

const (
	exeCmdName     = `exe`
	exeCmdSynopsis = `generate and compile (C)Python language bindings for Go, and make a standalone python executable with all the code.`
)

type exeCmd struct {
	*baseCmd

	exclude string
	user    string
	version string
	author  string
	email   string
	desc    string
	url     string
}

var _ subcommands.Command = (*exeCmd)(nil)

func NewExeCmd(ctx context.Context) *exeCmd {
	return &exeCmd{
		baseCmd: newBaseCmd(ctx),
	}
}

// Name returns the name of the command.
func (*exeCmd) Name() string { return exeCmdName }

// Synopsis returns a short string (less than one line) describing the command.
func (*exeCmd) Synopsis() string { return exeCmdSynopsis }

// Usage returns a long string explaining the command and giving usage information.
func (*exeCmd) Usage() string {
	return `usage: gopy exe <go-package-name> [other-go-package...]

Command exe generates and compiles (C)Python language bindings for a Go package,
including subdirectories, and generates a standalone python executable and
associated module packaging suitable for distribution.
It must provide suitable main function code.

If setup.py file does not yet exist in the target directory, then
it along with other default packaging files are created, using arguments.

Typically you create initial default versions of these files and then edit them, and
after that, only regenerate the go binding files.

The primary need for an exe instead of a pkg dynamic library is when the
main thread must be used for something other than running the
python interpreter, such as for a GUI library where the main thread must be
used for running the GUI event loop (e.g., GoGi).

When including multiple packages, list in order of increasing dependency,
and use -name arg to give appropriate name.

Example:
	gopy exe github.com/go-python/gopy/_examples/hi

`
}

// SetFlags adds the flags for this command to the specified set.
func (c *exeCmd) SetFlags(f *flag.FlagSet) {
	c.newFlags(f)

	f.StringVar(&c.exclude, "exclude", "", "comma-separated list of package names to exclude")
	f.StringVar(&c.version, "version", "0.1.0", "semantic version number")
	f.StringVar(&c.author, "author", "gopy", "author name")
	f.StringVar(&c.email, "email", "gopy@example.com", "author email")
	f.StringVar(&c.desc, "desc", "", "short description of project (long comes from README.md)")
	f.StringVar(&c.url, "url", "https://github.com/go-python/gopy", "home page for project")
}

// Execute executes the exe command and returns an [subcommands.ExitStatus].
func (c *exeCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...any) subcommands.ExitStatus {
	if f.NArg() == 0 {
		c.log.Error("gopy-exe: expect a fully qualified go package name as argument", slog.Any("args", f.Args()))
		f.Usage()
		return subcommands.ExitUsageError
	}

	cfg := NewBuildCfg()
	cfg.OutputDir = c.output
	cfg.Name = c.name
	cfg.Main = c.main
	cfg.VM = c.vm
	cfg.PkgPrefix = "" // doesn't make sense for exe
	cfg.RenameCase = c.rename
	cfg.Symbols = c.symbols
	cfg.NoWarn = c.noWarn
	cfg.NoMake = c.noMake
	cfg.DynamicLinking = c.dynamicLink
	cfg.BuildTags = c.buildTags

	exclude := c.exclude
	user := c.user
	version := c.version
	author := c.author
	email := c.email
	desc := c.desc
	url := c.url

	bind.NoWarn = cfg.NoWarn
	bind.NoMake = cfg.NoMake

	if cfg.Name == "" {
		path := f.Args()[0]
		_, cfg.Name = filepath.Split(path)
	}

	var err error
	cfg.OutputDir, err = genOutDir(cfg.OutputDir)
	if err != nil {
		c.log.Error("gopy-exe: get output directory", slog.Any("err", err))
		return subcommands.ExitFailure
	}

	setupfn := filepath.Join(cfg.OutputDir, "setup.py")

	if _, err = os.Stat(setupfn); os.IsNotExist(err) {
		err = GenPyPkgSetup(cfg, user, version, author, email, desc, url)
		if err != nil {
			c.log.Error("gopy-exe: generates python package setup files", slog.Any("err", err))
			return subcommands.ExitFailure
		}
	}

	defex := []string{"testdata", "internal", "python", "examples", "cmd"}
	excl := append(strings.Split(exclude, ","), defex...)
	exmap := make(map[string]struct{})
	for i := range excl {
		ex := strings.TrimSpace(excl[i])
		exmap[ex] = struct{}{}
	}

	cfg.OutputDir = filepath.Join(cfg.OutputDir, cfg.Name) // package must be in subdir
	cfg.OutputDir, err = genOutDir(cfg.OutputDir)
	if err != nil {
		c.log.Error("gopy-exe: generate output directory", slog.Any("err", err))
		return subcommands.ExitFailure
	}

	for _, path := range f.Args() {
		buildPkgRecurse(cfg.OutputDir, path, path, exmap, cfg.BuildTags)
	}
	if err := c.runBuild(ctx, bind.ModeExe, cfg); err != nil {
		c.log.Error("gopy-exe: build of package", slog.Any("err", err))
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
