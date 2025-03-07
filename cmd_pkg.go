// Copyright 2015 The go-python Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
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
	pkgCmdName     = `pkg`
	pkgCmdSynopsis = `generate and compile (C)Python language bindings for Go, and make a python package.`
)

type pkgCmd struct {
	*baseCmd

	exclude string
	user    string
	version string
	author  string
	email   string
	desc    string
	url     string
}

var _ subcommands.Command = (*pkgCmd)(nil)

func NewPkgCmd(ctx context.Context) *pkgCmd {
	return &pkgCmd{
		baseCmd: newBaseCmd(ctx),
	}
}

// Name returns the name of the command.
func (*pkgCmd) Name() string { return pkgCmdName }

// Synopsis returns the synopsis of the command.
func (*pkgCmd) Synopsis() string { return pkgCmdSynopsis }

// Usage returns a long string explaining the command and giving usage information.
func (*pkgCmd) Usage() string {
	return `gopy pkg <go-package-name> [other-go-package...]

Command pkg generates and compiles (C)Python language bindings for a Go package,
including subdirectories, and generates python module packaging suitable for distribution.

If setup.py file does not yet exist in the target directory, then it along
with other default packaging files are created, using arguments.

Typically you create initial default versions of these files and then edit them, and
after that, only regenerate the go binding files.

When including multiple packages, list in order of increasing dependency,
and use -name arg to give appropriate name.

Example:
	gopy pkg github.com/go-python/gopy/_examples/hi

`
}

// SetFlags adds the flags for this command to the specified set.
func (c *pkgCmd) SetFlags(f *flag.FlagSet) {
	c.newFlags(f)

	f.StringVar(&c.user, "user", "", "username on https://www.pypa.io/en/latest/ for package name suffix")
	f.StringVar(&c.version, "version", "0.1.0", "semantic version number")
	f.StringVar(&c.author, "author", "gopy", "author name")
	f.StringVar(&c.email, "email", "gopy@example.com", "author email")
	f.StringVar(&c.desc, "desc", "", "short description of project (long comes from README.md)")
	f.StringVar(&c.url, "url", "https://github.com/go-python/gopy", "home page for project")
	f.StringVar(&c.exclude, "exclude", "", "comma-separated list of package names to exclude")
}

// Execute executes the pkg command and returns an [subcommands.ExitStatus].
func (c *pkgCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...any) subcommands.ExitStatus {
	if len(f.Args()) == 0 {
		c.log.Error("gopy-build: expect a fully qualified go package name as argument", slog.Any("args", f.Args()))
		f.Usage()
		return subcommands.ExitUsageError
	}

	cfg := NewBuildCfg()
	cfg.OutputDir = c.output
	cfg.Name = c.name
	cfg.Main = c.main
	cfg.VM = c.vm
	cfg.PkgPrefix = c.packagePrefix
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
		c.log.Error("gopy-pkg: generate output directory", slog.String("output directory", cfg.OutputDir))
		return subcommands.ExitUsageError
	}

	setupfn := filepath.Join(cfg.OutputDir, "setup.py")

	if _, err = os.Stat(setupfn); os.IsNotExist(err) {
		err = GenPyPkgSetup(cfg, user, version, author, email, desc, url)
		if err != nil {
			c.log.Error("gopy-pkg: generate output directory", slog.String("output directory", cfg.OutputDir))
			return subcommands.ExitUsageError
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
		c.log.Error("gopy-pkg: generate output directory", slog.Any("err", err))
		return subcommands.ExitFailure
	}

	for _, path := range f.Args() {
		buildPkgRecurse(cfg.OutputDir, path, path, exmap, cfg.BuildTags)
	}

	if err := c.runBuild(ctx, bind.ModePkg, cfg); err != nil {
		c.log.Error("gopy-pkg: build of package", slog.Any("err", err))
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func buildPkgRecurse(odir, path, rootpath string, exmap map[string]struct{}, buildTags string) error {
	buildFirst := path == rootpath
	bpkg, err := loadPackage(path, buildFirst, buildTags)
	if err != nil {
		return fmt.Errorf("gopy-gen: go build / load of package failed with path=%q: %v", path, err)
	}
	gofiles := bpkg.GoFiles
	onego := ""
	if len(gofiles) == 1 {
		_, onego = filepath.Split(gofiles[0])
	}
	if len(gofiles) == 0 || (len(gofiles) == 1 && onego == "doc.go") {
		fmt.Printf("\n--- skipping dir with no go files or only doc.go: %s -- %s\n", path, gofiles)
		if len(gofiles) == 0 {
			// fmt.Printf("otherfiles: %v\nignorefiles: %v\n", bpkg.OtherFiles, bpkg.IgnoredFiles)
			if len(bpkg.OtherFiles) > 0 {
				gofiles = bpkg.OtherFiles
			} else if len(bpkg.IgnoredFiles) > 0 {
				gofiles = bpkg.IgnoredFiles
			} else {
				return nil // done
			}
		}
	} else {
		// fmt.Printf("gofiles: %s\n", gofiles)
		parsePackage(bpkg)
	}

	//	now try all subdirs
	dir, _ := filepath.Split(gofiles[0])
	drs := Dirs(dir)
	for _, dr := range drs {
		_, ex := exmap[dr]
		if ex || dr[0] == '.' || dr[0] == '_' || dr == "internal" {
			continue
		}
		sp := filepath.Join(path, dr)
		buildPkgRecurse(odir, sp, rootpath, exmap, buildTags)
	}
	return nil
}
