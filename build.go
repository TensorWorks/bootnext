//go:build never

package main

import (
	"flag"
	"os"

	module "github.com/tensorworks/go-build-helpers/pkg/module"
	validation "github.com/tensorworks/go-build-helpers/pkg/validation"
)

// Alias validation.ExitIfError() as check()
var check = validation.ExitIfError

func main() {

	// Parse our command-line flags
	doClean := flag.Bool("clean", false, "cleans build outputs")
	doRelease := flag.Bool("release", false, "builds executables for all target platforms")
	flag.Parse()

	// Disable CGO
	os.Setenv("CGO_ENABLED", "0")

	// Create a build helper for the Go module in the current working directory
	mod, err := module.ModuleInCwd()
	check(err)

	// Determine if we're cleaning the build outputs
	if *doClean == true {
		check(mod.CleanAll())
		os.Exit(0)
	}

	// Install the `go-winres` tool that we use for embedding manifest data in Windows builds
	check(mod.InstallGoTools([]string{
		"github.com/tc-hib/go-winres@v0.3.1",
	}))

	// Run `go generate` to invoke `go-winres`
	check(mod.Generate())

	// Determine if we're building our executables for just the host platform or for the full matrix of release platforms
	if *doRelease == false {
		check(mod.BuildBinariesForHost(module.DefaultBinDir, module.BuildOptions{Scheme: module.Undecorated}))
	} else {
		check(mod.BuildBinariesForMatrix(
			module.DefaultBinDir,

			module.BuildOptions{
				AdditionalFlags: []string{"-ldflags", "-s -w"},
				Scheme:          module.SuffixedFilenames,
			},

			module.BuildMatrix{
				Platforms:     []string{"linux", "windows"},
				Architectures: []string{"386", "amd64", "arm64"},
				Ignore:        []string{},
			},
		))
	}
}
