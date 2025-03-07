package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-python/gopy/bind"
	"golang.org/x/tools/imports"
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

func newBaseCmd(ctx context.Context) *baseCmd {
	return &baseCmd{
		log: FromContext(ctx),
	}
}

func (cmd *baseCmd) newFlags(f *flag.FlagSet) {
	f.StringVar(&cmd.vm, "vm", "python3", "path to python interpreter")
	f.StringVar(&cmd.output, "output", "", "output directory for bindings")
	f.StringVar(&cmd.name, "name", "", "name of output package (otherwise name of first package is used)")
	f.StringVar(&cmd.main, "main", "", "code string to run in the go main() function in the cgo library")
	f.StringVar(&cmd.packagePrefix, "package-prefix", ".", "custom package prefix used when generating import statements for generated package")
	f.BoolVar(&cmd.rename, "rename", false, "rename Go symbols to python PEP snake_case")
	f.BoolVar(&cmd.symbols, "symbols", true, "include symbols in output")
	f.BoolVar(&cmd.noWarn, "no-warn", false, "suppress warning messages, which may be expected")
	f.BoolVar(&cmd.noMake, "no-make", false, "do not generate a Makefile, e.g., when called from Makefile")
	f.BoolVar(&cmd.dynamicLink, "dynamic-link", false, "whether to link output shared library dynamically to Python")
	f.StringVar(&cmd.buildTags, "build-tags", "", "build tags to be passed to \"go build\"")
}

// runBuild calls genPkg and then executes commands to build the resulting files
// exe = executable mode to build an executable instead of a library
// mode = gen, build, pkg, exe
func (c *baseCmd) runBuild(ctx context.Context, mode bind.BuildMode, cfg *BuildCfg) (err error) {
	cfg.OutputDir, err = genOutDir(cfg.OutputDir)
	if err != nil {
		return fmt.Errorf("generate output directory: %w", err)
	}

	if err = genPkg(mode, cfg); err != nil {
		return fmt.Errorf("generate Python package: %w", err)
	}

	c.log.Info("gopy-build: building package", slog.Any("cmd", cfg.Cmd))

	buildname := cfg.Name + "_go"
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current directory: %w", err)
	}
	os.Chdir(cfg.OutputDir)
	defer os.Chdir(cwd)

	os.Remove(cfg.Name + ".c") // may fail, we don't care

	c.log.InfoContext(ctx, "format", slog.String("filename", cfg.Name+".go"))
	formatted, err := imports.Process(cfg.Name+".go", nil, &imports.Options{
		TabWidth:  8,
		TabIndent: true,
		Comments:  true,
		Fragment:  true,
	})
	if err != nil {
		return fmt.Errorf("format with imports: %w", err)
	}
	if err := os.WriteFile(cfg.Name+".go", formatted, 0o700); err != nil {
		return fmt.Errorf("write formatted data into %s: %w", cfg.Name+".go", err)
	}

	pycfg, err := bind.GetPythonConfig(cfg.VM)
	if err != nil {
		return fmt.Errorf("get Python config: %w", err)
	}
	c.log.DebugContext(ctx, "get Python config", slog.Any("pycfg", pycfg))

	if mode == bind.ModeExe {
		filename := buildname + ".h"
		of, err := os.Create(filename) // overwrite existing
		if err != nil {
			return fmt.Errorf("create %s file: %w", filename, err)
		}
		_, _ = of.WriteString("typedef uint8_t bool;\n")
		if err := of.Close(); err != nil {
			return fmt.Errorf("close %s file: %w", filename, err)
		}

		cmd := exec.CommandContext(ctx, cfg.VM, "build.py")
		c.log.InfoContext(ctx, "will fail, but needed to generate .c file", slog.String("command", strings.Join(cmd.Args, " ")))
		cmd.Run() // will fail, we don't care about errors

		args := []string{"build", "-mod=mod", "-buildmode=c-shared"}
		if cfg.BuildTags != "" {
			args = append(args, "-tags", cfg.BuildTags)
		}
		args = append(args, "-o", buildname+libExt, ".")

		cmd = exec.CommandContext(ctx, "go", args...)
		c.log.InfoContext(ctx, "execute command", slog.String("command", strings.Join(cmd.Args, " ")))
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("run %q command: output: %s: %w", strings.Join(cmd.Args, " "), string(output), err)
		}

		cmd = exec.Command(cfg.VM, "build.py")
		c.log.InfoContext(ctx, "execute command", slog.String("command", strings.Join(cmd.Args, " ")))
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("run %q command: output: %s: %w", strings.Join(cmd.Args, " "), string(output), err)
		}

		err = os.Remove(cfg.Name + "_go" + libExt)
		if err != nil {
			return fmt.Errorf("remove %s file: %w", cfg.Name+"_go"+libExt, err)
		}

		cmd = exec.Command("go", "build", "-mod=mod")
		c.log.InfoContext(ctx, "execute command", slog.String("command", strings.Join(cmd.Args, " ")))
		if cfg.BuildTags != "" {
			args = append(args, "-tags", cfg.BuildTags)
		}
		args = append(args, "-o", "py"+cfg.Name)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("run %q command: output: %s: %w", strings.Join(cmd.Args, " "), string(output), err)
		}

	} else {
		buildLib := buildname + libExt
		extext := libExt
		if runtime.GOOS == "windows" {
			extext = ".pyd"
		}
		if pycfg.ExtSuffix != "" {
			extext = pycfg.ExtSuffix
		}
		modlib := "_" + cfg.Name + extext

		// build the go shared library upfront to generate the header
		// needed by our generated cpython code
		args := []string{"build", "-mod=mod", "-buildmode=c-shared"}
		if cfg.BuildTags != "" {
			args = append(args, "-tags", cfg.BuildTags)
		}
		if !cfg.Symbols {
			// These flags will omit the various symbol tables, thereby
			// reducing the final size of the binary. From https://golang.org/cmd/link/
			// -s Omit the symbol table and debug information
			// -w Omit the DWARF symbol table
			args = append(args, "-ldflags=-s -w")
		}
		args = append(args, "-o", buildLib, ".")

		cmd := exec.CommandContext(ctx, "go", args...)
		c.log.InfoContext(ctx, "execute command", slog.String("command", strings.Join(cmd.Args, " ")))
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("run %q command: output: %s: %w", strings.Join(cmd.Args, " "), string(output), err)
		}
		// update the output name to the one with the ABI extension
		args[len(args)-2] = modlib
		// we don't need this initial lib because we are going to relink
		os.Remove(buildLib)

		// generate C code
		cmd = exec.Command(cfg.VM, "build.py")
		c.log.InfoContext(ctx, "execute command", slog.String("command", strings.Join(cmd.Args, " ")))
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("run %q command: output: %s: %w", strings.Join(cmd.Args, " "), string(output), err)
		}

		if bind.WindowsOS {
			c.log.InfoContext(ctx, "Doing windows sed hack to fix declspec for PyInit")
			fname := cfg.Name + ".c"
			raw, err := os.ReadFile(fname)
			if err != nil {
				return fmt.Errorf("could not read %s: %w", fname, err)
			}
			raw = bytes.ReplaceAll(raw, []byte(" PyInit_"), []byte(" __declspec(dllexport) PyInit_"))
			err = os.WriteFile(fname, raw, 0o644)
			if err != nil {
				return fmt.Errorf("could not apply sed hack to fix PyInit: %w", err)
			}
		}

		cflags := strings.Fields(strings.TrimSpace(pycfg.CFlags))
		cflags = append(cflags, "-fPIC", "-O3", "-ffast-math")
		if include, exists := os.LookupEnv("GOPY_INCLUDE"); exists {
			cflags = append(cflags, "-I"+filepath.ToSlash(include))
		}
		if oldcflags, exists := os.LookupEnv("CGO_CFLAGS"); exists {
			cflags = append(cflags, oldcflags)
		}
		var ldflags []string
		if cfg.DynamicLinking {
			ldflags = strings.Fields(strings.TrimSpace(pycfg.LdDynamicFlags))
		} else {
			ldflags = strings.Fields(strings.TrimSpace(pycfg.LdFlags))
		}
		if !cfg.Symbols {
			ldflags = append(ldflags, "-s")
		}
		if lib, exists := os.LookupEnv("GOPY_LIBDIR"); exists {
			ldflags = append(ldflags, "-L"+filepath.ToSlash(lib))
		}
		if libname, exists := os.LookupEnv("GOPY_PYLIB"); exists {
			ldflags = append(ldflags, "-l"+filepath.ToSlash(libname))
		}
		if oldldflags, exists := os.LookupEnv("CGO_LDFLAGS"); exists {
			ldflags = append(ldflags, oldldflags)
		}

		removeEmpty := func(src []string) []string {
			o := make([]string, 0, len(src))
			for _, v := range src {
				if v == "" {
					continue
				}
				o = append(o, v)
			}
			return o
		}

		cflags = removeEmpty(cflags)
		ldflags = removeEmpty(ldflags)

		cflagsEnv := fmt.Sprintf("CGO_CFLAGS=%s", strings.Join(cflags, " "))
		ldflagsEnv := fmt.Sprintf("CGO_LDFLAGS=%s", strings.Join(ldflags, " "))

		env := os.Environ()
		env = append(env, cflagsEnv)
		env = append(env, ldflagsEnv)

		fmt.Println(cflagsEnv)
		fmt.Println(ldflagsEnv)

		// build extension with go + c
		cmd = exec.Command("go", args...)
		c.log.InfoContext(ctx, "execute command", slog.String("command", strings.Join(cmd.Args, " ")))
		cmd.Env = env
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("run %q command: output: %s: %w", strings.Join(cmd.Args, " "), string(output), err)
		}
	}

	return err
}
