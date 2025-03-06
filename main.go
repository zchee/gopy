// Copyright 2015 The go-python Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/go-faster/errors"
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
	commander.Explain = func(w io.Writer) {
		fmt.Fprintf(w, `gopy generates a CPython extension module from a go package.`)
	}
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")
	subcommands.Register(&gopyCmd{}, "gopy")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	os.Exit(int(subcommands.Execute(ctx)))
}

func (c *gopyCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...any) subcommands.ExitStatus {
	gopyMakeCmdGen()
	gopyMakeCmdBuild()
	gopyMakeCmdPkg()
	gopyMakeCmdExe()
	appArgs := f.Args()
	if status := c.Execute(ctx, f, appArgs); !isSuccess(status) {
		fmt.Errorf("error dispatching command: %v", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func run(ctx context.Context, args []string) error {
	// app := &commander.Command{
	// 	UsageLine: "gopy",
	// 	Subcommands: []*commander.Command{
	// 		gopyMakeCmdGen(),
	// 		gopyMakeCmdBuild(),
	// 		gopyMakeCmdPkg(),
	// 		gopyMakeCmdExe(),
	// 	},
	// 	Flag: *flag.NewFlagSet("gopy", flag.ExitOnError),
	// }

	// err := app.Flag.Parse(args)
	// if err != nil {
	// 	return fmt.Errorf("could not parse flags: %v", err)
	// }
	//
	// appArgs := app.Flag.Args()
	// err = app.Dispatch(ctx, appArgs)
	// if err != nil {
	// 	return fmt.Errorf("error dispatching command: %v", err)
	// }
	return nil
}

func copyCmd(src, dst string) error {
	srcf, err := os.Open(src)
	if err != nil {
		return errors.Wrap(err, "could not open source for copy")
	}
	defer srcf.Close()

	os.MkdirAll(path.Dir(dst), 0o755)

	dstf, err := os.Create(dst)
	if err != nil {
		return errors.Wrap(err, "could not create destination for copy")
	}
	defer dstf.Close()

	_, err = io.Copy(dstf, srcf)
	if err != nil {
		return errors.Wrap(err, "could not copy bytes to destination")
	}

	err = dstf.Sync()
	if err != nil {
		return errors.Wrap(err, "could not synchronize destination")
	}

	return dstf.Close()
}
