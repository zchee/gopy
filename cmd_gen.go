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
	genCmdName     = `gen`
	genCmdSynopsis = `generate (C)Python language bindings for Go.`
)

type genCmd struct {
	*baseCmd
}

var _ subcommands.Command = (*genCmd)(nil)

func NewGenCmd(ctx context.Context) *genCmd {
	return &genCmd{
		baseCmd: newBaseCmd(ctx),
	}
}

// Name returns the name of the command.
func (*genCmd) Name() string { return genCmdName }

// Synopsis returns the synopsis of the command.
func (*genCmd) Synopsis() string { return genCmdSynopsis }

// Usage returns a long string explaining the command and giving usage information.
func (*genCmd) Usage() string {
	return `usage: gopy gen <go-package-name> [other-go-package...]

Command gen generates (C)Python language bindings for Go package(s).

Example:
gopy gen github.com/go-python/gopy/_examples/hi

`
}

// SetFlags adds the flags for this command to the specified set.
func (c *genCmd) SetFlags(f *flag.FlagSet) {
	c.newFlags(f)
}

// Execute executes the gen command and returns an [subcommands.ExitStatus].
func (c *genCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...any) subcommands.ExitStatus {
	if f.NArg() == 0 {
		c.log.Error("gopy-gen: expect a fully qualified go package name as argument", slog.Any("args", f.Args()))
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

	if cfg.VM == "" {
		cfg.VM = "python3"
	}

	bind.NoWarn = cfg.NoWarn
	bind.NoMake = cfg.NoMake

	for _, path := range f.Args() {
		bpkg, err := loadPackage(path, true, cfg.BuildTags) // build first
		if err != nil {
			c.log.Error("gopy-gen: go build / load of package", slog.String("path", path), slog.Any("err", err))
			return subcommands.ExitFailure
		}
		pkg, err := parsePackage(bpkg)
		if err != nil {
			c.log.Error("gopy-gen: parse package", slog.Any("bpkg", bpkg), slog.Any("err", err))
			return subcommands.ExitFailure
		}
		if cfg.Name == "" {
			cfg.Name = pkg.Name()
		}
	}

	if err := genPkg(bind.ModeGen, cfg); err != nil {
		c.log.Error("gopy-gen: generate package", slog.Any("err", err))
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
