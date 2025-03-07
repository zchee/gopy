// Copyright 2019 The go-python Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin
// +build darwin

package main

import (
	"bytes"
	"os/exec"
)

const (
	// libExt = ".dylib"  // theoretically should be this but python only recognizes .so
	libExt = ".so"
)

var extraGccArgs = "-dynamiclib"

func init() {
	xcrunCmd := exec.Command("/usr/bin/xcrun", "--show-sdk-path")
	output, err := xcrunCmd.CombinedOutput()
	if err != nil {
		panic(err)
	}
	extraGccArgs += " -sysroot " + string(bytes.TrimSpace(output))
}
