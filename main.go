// Copyright 2015 The go-python Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"path"

	"github.com/google/subcommands"

	"github.com/go-python/gopy/bind"
)

// BuildCfg contains command options and binding generation options
type BuildCfg struct {
	bind.BindCfg

	// include symbols in output
	Symbols bool
	// suppress warning messages, which may be expected
	NoWarn bool
	// do not generate a Makefile, e.g., when called from Makefile
	NoMake bool
	// link resulting library dynamically
	DynamicLinking bool
	// BuildTags to be passed into `go build`.
	BuildTags string
}

// NewBuildCfg returns a newly constructed build config
func NewBuildCfg() *BuildCfg {
	var cfg BuildCfg
	cfg.Cmd = argStr()
	return &cfg
}

func main() {
	commander := subcommands.NewCommander(flag.CommandLine, path.Base(os.Args[0]))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	ctx = IntoContext(ctx, log)

	commander.Register(commander.FlagsCommand(), "")
	commander.Register(commander.CommandsCommand(), "")
	commander.Register(commander.HelpCommand(), "")
	commander.Register(NewGenCmd(ctx), "gopy")
	commander.Register(NewBuildCmd(ctx), "gopy")
	commander.Register(NewExeCmd(ctx), "gopy")
	commander.Register(NewPkgCmd(ctx), "gopy")

	flag.Parse()

	os.Exit(int(commander.Execute(ctx)))
}
