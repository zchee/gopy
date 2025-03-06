package main

import (
	"context"
	"flag"
	"log/slog"

	"github.com/google/subcommands"
)

type baseCmd struct {
	log *slog.Logger

	vm            string
	output        string
	name          string
	main          string
	packagePrefix string
	rename        bool
	symbols       bool
	noWarn        bool
	noMake        bool
	dynamicLink   bool
	buildTags     string
}

func (cmd *baseCmd) newFlags(f *flag.FlagSet) {
	f.StringVar(&cmd.vm, "vm", "python", "path to python interpreter")
	f.StringVar(&cmd.output, "output", "", "output directory for bindings")
	f.StringVar(&cmd.name, "name", "", "name of output package (otherwise name of first package is used)")
	f.StringVar(&cmd.main, "main", "", "code string to run in the go main() function in the cgo library")
	f.StringVar(&cmd.packagePrefix, "package-prefix", ".", "custom package prefix used when generating import statements for generated package")
	f.BoolVar(&cmd.rename, "rename", false, "rename Go symbols to python PEP snake_case")
	f.BoolVar(&cmd.symbols, "symbols", true, "include symbols in output")
	f.BoolVar(&cmd.noWarn, "no-warn", false, "suppress warning messages, which may be expected")
	f.BoolVar(&cmd.noMake, "no-make", false, "do not generate a Makefile, e.g., when called from Makefile")
	f.BoolVar(&cmd.dynamicLink, "dynamic-link", false, "whether to link output shared library dynamically to Python")
	f.StringVar(&cmd.buildTags, "build-tags", "", "build tags to be passed to `go build`")
}

func newBaseCmd(ctx context.Context) *baseCmd {
	return &baseCmd{
		log: FromContext(ctx),
	}
}

type gopyCmd struct {
	*baseCmd
}

var _ subcommands.Command = (*gopyCmd)(nil)

func NewGoPyCmd(ctx context.Context) *gopyCmd {
	return &gopyCmd{
		baseCmd: newBaseCmd(ctx),
	}
}

func (*gopyCmd) Name() string { return `gopy` }
func (*gopyCmd) Synopsis() string {
	return `gopy generates a CPython extension module from a go package`
}
func (*gopyCmd) Usage() string          { return `gopy [flags] <Python plugin Go files>` }
func (*gopyCmd) SetFlags(*flag.FlagSet) {}

func (*gopyCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...any) subcommands.ExitStatus {
	subcommands.Register(NewGenCmd(ctx), "")
	subcommands.Register(NewBuildCmd(ctx), "")
	subcommands.Register(NewPkgCmd(ctx), "")
	subcommands.Register(NewPkgCmd(ctx), "")

	return subcommands.ExitSuccess
}

type genCmd struct {
	*baseCmd
}

var _ subcommands.Command = (*gopyCmd)(nil)

func NewGenCmd(ctx context.Context) *genCmd {
	return &genCmd{
		baseCmd: newBaseCmd(ctx),
	}
}

func (*genCmd) Name() string               { return `gen` }
func (*genCmd) Synopsis() string           { return `generate (C)Python language bindings for Go` }
func (*genCmd) Usage() string              { return `gen <go-package-name> [other-go-package...]` }
func (c *genCmd) SetFlags(f *flag.FlagSet) { c.newFlags(f) }

func (c *genCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...any) subcommands.ExitStatus {
	return subcommands.ExitSuccess
}

type buildCmd struct {
	*baseCmd

	user    string
	version string
	author  string
	email   string
	desc    string
	url     string
}

var _ subcommands.Command = (*gopyCmd)(nil)

func NewBuildCmd(ctx context.Context) *buildCmd {
	return &buildCmd{
		baseCmd: newBaseCmd(ctx),
	}
}

func (*buildCmd) Name() string { return `gen` }
func (*buildCmd) Synopsis() string {
	return `
generate and compile (C)Python language bindings for Go.

gopy generates a CPython extension module from a go package.
`
}
func (*buildCmd) Usage() string { return `gen <go-package-name> [other-go-package...]` }
func (c *buildCmd) SetFlags(f *flag.FlagSet) {
	c.newFlags(f)

	f.StringVar(&c.user, "user", "", "username on https://www.pypa.io/en/latest/ for package name suffix")
	f.StringVar(&c.version, "version", "0.1.0", "semantic version number -- can use e.g., git to get this from tag and pass as argument")
	f.StringVar(&c.author, "author", "gopy", "author name")
	f.StringVar(&c.email, "email", "gopy@example.com", "author email")
	f.StringVar(&c.desc, "desc", "", "short description of project (long comes from README.md)")
	f.StringVar(&c.url, "url", "https://github.com/go-python/gopy", "home page for project")
}

func (c *buildCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...any) subcommands.ExitStatus {
	return subcommands.ExitSuccess
}

type exeCmd struct {
	*baseCmd
}

var _ subcommands.Command = (*exeCmd)(nil)

func NewExeCmd(ctx context.Context) *exeCmd {
	return &exeCmd{
		baseCmd: newBaseCmd(ctx),
	}
}

func (*exeCmd) Name() string { return `pkg` }
func (*exeCmd) Synopsis() string {
	return `
generate and compile (C)Python language bindings for Go, and make a standalone python executable with all the code -- must provide suitable main function code

exe generates and compiles (C)Python language bindings for a Go package, including subdirectories, and generates a standalone python executable and associated module packaging suitable for distribution.  if setup.py file does not yet exist in the target directory, then it along with other default packaging files are created, using arguments.  Typically you create initial default versions of these files and then edit them, and after that, only regenerate the go binding files.

The primary need for an exe instead of a pkg dynamic library is when the main thread must be used for something other than running the python interpreter, such as for a GUI library where the main thread must be used for running the GUI event loop (e.g., GoGi).

When including multiple packages, list in order of increasing dependency, and use -name arg to give appropriate name.

ex:
 $ gopy exe [options] <go-package-name> [other-go-package...]
 $ gopy exe github.com/go-python/gopy/_examples/hi
`
}
func (*exeCmd) Usage() string          { return `pkg <go-package-name> [other-go-package...]` }
func (*exeCmd) SetFlags(*flag.FlagSet) {}

func (c *exeCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...any) subcommands.ExitStatus {
	return subcommands.ExitSuccess
}

type pkgCmd struct {
	*baseCmd

	user    string
	version string
	author  string
	email   string
	desc    string
	url     string
	noMake  bool
	exclude string
}

var _ subcommands.Command = (*gopyCmd)(nil)

func NewPkgCmd(ctx context.Context) *pkgCmd {
	return &pkgCmd{
		baseCmd: newBaseCmd(ctx),
	}
}

func (*pkgCmd) Name() string { return `exe` }
func (*pkgCmd) Synopsis() string {
	return `
generate and compile (C)Python language bindings for Go, and make a python package.

pkg generates and compiles (C)Python language bindings for a Go package, including subdirectories, and generates python module packaging suitable for distribution.  if setup.py file does not yet exist in the target directory, then it along with other default packaging files are created, using arguments.  Typically you create initial default versions of these files and then edit them, and after that, only regenerate the go binding files.

When including multiple packages, list in order of increasing dependency, and use -name arg to give appropriate name.

ex:
 $ gopy pkg [options] <go-package-name> [other-go-package...]
 $ gopy pkg github.com/go-python/gopy/_examples/hi
`
}
func (*pkgCmd) Usage() string { return `exe <go-package-name> [other-go-package...]` }
func (c *pkgCmd) SetFlags(f *flag.FlagSet) {
	c.newFlags(f)

	f.StringVar(&c.user, "user", "", "username on https://www.pypa.io/en/latest/ for package name suffix")
	f.StringVar(&c.version, "version", "0.1.0", "semantic version number -- can use e.g., git to get this from tag and pass as argument")
	f.StringVar(&c.author, "author", "gopy", "author name")
	f.StringVar(&c.email, "email", "gopy@example.com", "author email")
	f.StringVar(&c.desc, "desc", "", "short description of project (long comes from README.md)")
	f.StringVar(&c.url, "url", "https://github.com/go-python/gopy", "home page for project")
	f.StringVar(&c.exclude, "exclude", "", "comma-separated list of package names to exclude")
}

func (c *exeCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...any) subcommands.ExitStatus {
	return subcommands.ExitSuccess
}

// isSuccess reports whether the status is [subcommands.ExitSuccess].
func isSuccess(status subcommands.ExitStatus) bool {
	return status == subcommands.ExitSuccess
}
