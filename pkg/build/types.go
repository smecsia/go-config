package build

import (
	"fmt"
	"github.com/smecsia/go-utils/pkg/git"
	"strings"
)

type Generator func() error

// Cmd executable command definition
type Cmd struct {
	Command string
	Wd      string
	Env     []string
}

// Context build context and config
type Context struct {
	OutDir       string     `yaml:"outDir,omitempty" env:"OUT_DIR" default:"bin"`
	Version      string     `yaml:"version,omitempty" env:"VERSION" default:""`
	Platforms    []Platform `yaml:"platforms,omitempty"`
	Targets      []Target   `yaml:"targets,omitempty"`

	// env-only fields
	GitAuthor       string `yaml:"-" default:"bambooagent" env:"GIT_AUTHOR"`
	GitBranch       string `yaml:"-" default:"master" env:"GIT_BRANCH"`
	GitRemote       string `yaml:"-" default:"origin" env:"GIT_REMOTE"`
	Parallel        string `yaml:"-" default:"true" env:"PARALLEL"`
	SkipTests       string `yaml:"-" default:"false" env:"SKIP_TESTS"`
	Verbose         string `yaml:"-" default:"false" env:"VERBOSE"`
	FilterTargets   string `yaml:"-" default:"-" env:"TARGETS"`
	FilterPlatforms string `yaml:"-" default:"-" env:"PLATFORMS"`

	// init-only private fields
	configFilePath string
	git            git.Git
}

func (ctx *Context) SetConfigFilePath(path string) {
	ctx.configFilePath = path
}

func (ctx *Context) GetConfigFilePath() string {
	return ctx.configFilePath
}

func (ctx *Context) Init() error {
	ctx.git = git.NewWithCfg(ctx.GitRoot(), ctx.GitAuthor, ctx.GitBranch, ctx.GitRemote)
	return nil
}

// Platform defines platform to run build for
type Platform struct {
	GOOS   string `yaml:"os,omitempty"`
	GOARCH string `yaml:"arch,omitempty"`
}

// Target defines target to run build for
type Target struct {
	Name string `yaml:"name,omitempty"`
	Path string `yaml:"path,omitempty"`
}

// Bundle defines result of tarball build
type Bundle struct {
	Target       Target
	Platform     Platform
	BinaryFile   string
	ChecksumFile string
}

// IsParallel returns true if concurrent execution required
func (ctx *Context) IsParallel() bool {
	return ctx.Parallel == "true"
}

// IsSkipTests returns true if tests must be skipped
func (ctx *Context) IsSkipTests() bool {
	return ctx.SkipTests == "true"
}

// IsVerbose returns true if tests must be skipped
func (ctx *Context) IsVerbose() bool {
	return ctx.Verbose == "true"
}

// ActivePlatforms returns list of active platforms
func (ctx *Context) ActivePlatforms() []Platform {
	if ctx.FilterPlatforms == "-" {
		return ctx.Platforms
	}
	res := make([]Platform, 0)
	usePlatforms := strings.Split(ctx.FilterPlatforms, ",")
	for _, platform := range ctx.Platforms {
		for _, usePlatform := range usePlatforms {
			platformParts := strings.Split(usePlatform, ":")
			if platformParts[0] == platform.GOOS && platformParts[1] == platform.GOARCH {
				res = append(res, platform)
			}
		}
	}
	fmt.Println("Active platforms: ", res)
	return res
}

// ActiveTargets returns true if concurrent execution required
func (ctx *Context) ActiveTargets() []Target {
	if ctx.FilterTargets == "-" {
		return ctx.Targets
	}
	res := make([]Target, 0)
	useTargets := strings.Split(ctx.FilterTargets, ",")
	for _, target := range ctx.Targets {
		for _, useTarget := range useTargets {
			if useTarget == target.Name {
				res = append(res, target)
			}
		}
	}
	fmt.Println("Active targets: ", res)
	return res
}
