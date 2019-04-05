// +build mage

package main

import (
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/magefile/mage/mg" // mg contains helpful utility functions, like Deps
	"github.com/smecsia/go-utils/pkg/build"
	"github.com/smecsia/go-utils/pkg/util"
	"golang.org/x/sync/errgroup"
	"os"
	"strings"
	/**/)

// Default target to run when none is specified
// If not set, running mage will list available targets
// var Default = Build

type Build mg.Namespace
type Tests mg.Namespace
type Generate mg.Namespace
type Version mg.Namespace
type Git mg.Namespace
type Publish mg.Namespace

var (
	Ctx        = build.Init("./build.yaml", util.DefaultConsoleReader)
	Generators = map[string]build.Generator{
		"templates": nsGenerate.Templates,
	}
)

var (
	nsGenerate = Generate{}
	nsBuild    = Build{}
	nsTests    = Tests{}
	nsVersion  = Version{}
	nsGit      = Git{}
	nsPublish  = Publish{}
)

// --------------------------------------
// Git targets

// CheckStatus Checks current status and fails if there are changes
func (Git) CheckStatus() error {
	return Ctx.GitCheckWorkTree()
}

// Commit Makes commit of current local changes on the build agent behalf
func (Git) Commit() error {
	return Ctx.GitCommitAndPush("Automatically generated commit")
}

// --------------------------------------
// Test targets

// Test Runs all types of tests
func Test() error {
	mg.Deps(nsTests.Unit)
	return nil
}

// Unit Run unit tests
func (Tests) Unit() error {
	if !Ctx.IsSkipTests() {
		verbose := ""
		if Ctx.IsVerbose() {
			verbose = " -v"
		}
		return Ctx.RunCmd(build.Cmd{Command: "go test" + verbose + " ./...", Env: Ctx.CurrentPlatform().GoEnv()})
	}
	return nil
}

// --------------------------------------
// Build targets

// All Run everything: install deps, generate clients, run all builds for all platforms
func (b Build) All() error {
	mg.Deps(b.InstallDeps)
	mg.Deps(b.Build)
	mg.Deps(nsGit.CheckStatus)
	return b.Bundles()
}

// Build Builds all plugins at once
func (Build) Build() error {
	mg.Deps(nsGenerate.All)
	mg.Deps(nsTests.Unit)
	return Ctx.ForAllTargets(func(target build.Target) error {
		return Ctx.ForAllPlatforms(func(platform build.Platform) error {
			return Ctx.Build(target, platform)
		})
	})
}

// DeploymentsAll Builds deployments plugin for all platforms at once
func (Build) DeploymentsCmdAll() error {
	return Ctx.ForAllPlatforms(func(platform build.Platform) error {
		return Ctx.Build(Ctx.Target("deployments"), platform)
	})
}

// Deployments Builds deployments plugin for current platform only
func (Build) DeploymentsCmd() error {
	mg.Deps(nsGenerate.Templates)
	return Ctx.Build(Ctx.Target("deployments"), Ctx.CurrentPlatform())
}

// BuildAll Builds build plugin for all platforms at once
func (Build) BuildCmdAll() error {
	return Ctx.ForAllPlatforms(func(platform build.Platform) error {
		return Ctx.Build(Ctx.Target("build"), platform)
	})
}

// InstallDeps Installs dependencies
func (Build) InstallDeps() error {
	fmt.Println("Installing Deps...")
	cmd := "bin/dep"
	if _, err := os.Stat(cmd); os.IsNotExist(err) {
		cmd = "dep"
	}
	return Ctx.RunCmd(build.Cmd{Command: fmt.Sprintf("%s ensure -v", cmd)})
}

// Clean Cleans up output directory
func (Build) Clean() error {
	fmt.Println("Cleaning...")
	return os.RemoveAll(Ctx.OutDir)
}

// Bundles Build tarballs out of binaries
func (b Build) Bundles() error {
	fmt.Println("Bundling tarballs...")
	var eg errgroup.Group
	for _, target := range Ctx.ActiveTargets() {
		for _, platform := range Ctx.Platforms {
			t := target
			p := platform
			bundle := Ctx.Bundle(t, p)
			buildTarball := func() error {
				if err := Ctx.BuildBundle(bundle); err != nil {
					return err
				}
				return nil
			}
			if Ctx.IsParallel() {
				eg.Go(buildTarball)
			} else if err := buildTarball(); err != nil {
				return err
			}
		}
	}
	return eg.Wait()
}

// --------------------------------------
// Publish


// --------------------------------------
// Generators

// Invoke all generators and generate all necessary clients
func (Generate) All() error {
	var eg errgroup.Group
	for _, generator := range Generators {
		g := generator
		if Ctx.IsParallel() {
			eg.Go(g)
		} else if err := g(); err != nil {
			return err
		}
	}
	return eg.Wait()
}

// Templates Generate templates.tpl.go file from /templates folder
func (Generate) Templates() error {
	fmt.Println("Regenerating templates.tpl.go...")
	templatesDir := Ctx.Path("templates")
	if paths, err := Ctx.ListAllSubDirs(templatesDir); err != nil {
		return err
	} else {
		return Ctx.RunCmd(build.Cmd{Command: fmt.Sprintf(
			"go run %s -nometadata -pkg render -prefix %s/ -o %s %s",
			Ctx.VendorPath("github.com/go-bindata/go-bindata/go-bindata"),
			templatesDir, Ctx.Path("pkg/render/templates.tpl.go"), strings.Join(paths, " ")),
			Env: Ctx.CurrentPlatform().GoEnv(), Wd: templatesDir})
	}
}

// --------------------------------------
// Version targets

// BumpMajor Bump major version
func (Version) BumpMajor() error {
	v := Ctx.SemVer().IncMajor()
	return Ctx.SetVersionInConfig(v.String())
}

// BumpMinor Bump minor version
func (Version) BumpMinor() error {
	v := Ctx.SemVer().IncMinor()
	return Ctx.SetVersionInConfig(v.String())
}

// BumpPatch Bump patch version
func (Version) BumpPatch() error {
	v := Ctx.SemVer().IncPatch()
	return Ctx.SetVersionInConfig(v.String())
}

// Print Prints full version (current + git hash)
func (Version) Print() error {
	fmt.Println(Ctx.FullVersion())
	return nil
}

// Set Sets new version
func (Version) Set() error {
	if v, err := semver.NewVersion(os.Getenv("VERSION")); err != nil {
		return err
	} else {
		return Ctx.SetVersionInConfig(v.String())
	}
}
