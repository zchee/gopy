// Copyright 2015 The go-python Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"log/slog"

	"github.com/go-python/gopy/bind"
	"github.com/google/subcommands"
)

const (
	buildCmdName     = `build`
	buildCmdSynopsis = `generate and compile (C)Python language bindings for Go.`
)

type buildCmd struct {
	*baseCmd
}

var _ subcommands.Command = (*buildCmd)(nil)

func NewBuildCmd(ctx context.Context) *buildCmd {
	return &buildCmd{
		baseCmd: newBaseCmd(ctx),
	}
}

// Name returns the name of the command.
func (c *buildCmd) Name() string { return buildCmdName }

// Synopsis returns a short string (less than one line) describing the command.
func (c *buildCmd) Synopsis() string { return buildCmdSynopsis }

// Usage returns a long string explaining the command and giving usage information.
func (c *buildCmd) Usage() string {
	return `usage: gopy build <go-package-name> [other-go-package...]

Command build generates and compiles (C)Python language bindings for Go package(s).

Example:
	gopy build github.com/go-python/gopy/_examples/hi

`
}

// SetFlags adds the flags for this command to the specified set.
func (c *buildCmd) SetFlags(f *flag.FlagSet) {
	c.newFlags(f)
}

// Execute executes the build command and returns an [subcommands.ExitStatus].
func (c *buildCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...any) subcommands.ExitStatus {
	if f.NArg() == 0 {
		c.log.Error("gopy build: expect a fully qualified go package name as argument", slog.Any("args", f.Args()))
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

	bind.NoWarn = cfg.NoWarn
	bind.NoMake = cfg.NoMake

	for _, path := range f.Args() {
		bpkg, err := loadPackage(path, true, cfg.BuildTags) // build first
		if err != nil {
			c.log.Error("gopy build: load of package", slog.String("path", path), slog.Any("err", err))
			return subcommands.ExitFailure
		}
		pkg, err := parsePackage(bpkg)
		if err != nil {
			c.log.Error("gopy build: parse of package", slog.Any("err", err))
			return subcommands.ExitFailure
		}
		if cfg.Name == "" {
			cfg.Name = pkg.Name()
		}
	}

	if err := c.runBuild(ctx, "build", cfg); err != nil {
		c.log.Error("gopy build: build of package", slog.Any("err", err))
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
