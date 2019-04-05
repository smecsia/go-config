package build

import (
	"crypto/sha256"
	"fmt"
	"github.com/pkg/errors"
	"go/build"
	"golang.org/x/sync/errgroup"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	GoBuildEnv = []string{
		"GOPATH=" + build.Default.GOPATH,
	}
)

// CurrentPlatform returns current platform
func (ctx Context) CurrentPlatform() Platform {
	return Platform{
		GOOS:   runtime.GOOS,
		GOARCH: runtime.GOARCH,
	}
}

// BuildAllPlatforms runs build for all platforms for target
func (ctx *Context) BuildAllPlatforms(target Target) error {
	var eg errgroup.Group
	for _, platform := range ctx.Platforms {
		p := platform
		if ctx.IsParallel() {
			eg.Go(func() error {
				return ctx.Build(target, p)
			})
		} else if err := ctx.Build(target, p); err != nil {
			return err
		}
	}
	return eg.Wait()
}

// ForAllPlatforms runs build for all platforms for target
func (ctx *Context) ForAllPlatforms(action func(platform Platform) error) error {
	var eg errgroup.Group
	for _, platform := range ctx.ActivePlatforms() {
		p := platform
		buildFnc := func() error {
			return action(p)
		}
		if ctx.IsParallel() {
			eg.Go(buildFnc)
		} else if err := buildFnc(); err != nil {
			return err
		}
	}
	return eg.Wait()
}

// ForAllTargets execute some action for each target
func (ctx *Context) ForAllTargets(action func(target Target) error) error {
	var eg errgroup.Group
	for _, target := range ctx.ActiveTargets() {
		t := target
		actionFor := func() error {
			return action(t)
		}
		if ctx.IsParallel() {
			eg.Go(actionFor)
		} else if err := actionFor(); err != nil {
			return err
		}
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

// Build runs go build for a certain target and platform
func (ctx *Context) Build(target Target, platform Platform) error {
	fmt.Println(fmt.Sprintf("Building %s for %s...", target, platform))

	outFile := ctx.OutFile(target, platform)

	err := os.MkdirAll(filepath.Dir(outFile), os.ModePerm)
	if err != nil {
		return err
	}

	return ctx.RunCmd(Cmd{Env: platform.GoEnv(),
		Command: "go build -ldflags \"-X main.Version=" + ctx.FullVersion() + "\" -o \"" + outFile + "\" \"" + ctx.SourceFile(target) + "\""})
}

// RunCmd runs any command in the sub shell
func (ctx *Context) RunCmd(cmd Cmd) error {
	run := exec.Command("bash", "-c", cmd.Command)
	runString := fmt.Sprintf("%s \"%s\"", run.Path, strings.Join(run.Args[1:], "\" \""))
	if cmd.Wd == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		run.Dir = cwd
	} else {
		run.Dir = cmd.Wd
	}
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	run.Env = os.Environ()
	for _, env := range cmd.Env {
		run.Env = append(run.Env, env)
	}
	fmt.Println(fmt.Sprintf("Executing '%s'", runString))
	err := run.Run()
	if err != nil {
		return errors.Wrapf(err, "Failed to execute '%s'", runString)
	}
	return nil
}

// GenerateSwaggerClient generates swagger client from URL into relative path
func (ctx *Context) GenerateSwaggerClient(relPath string, swaggerURL string, appName string) error {
	fmt.Println(fmt.Sprintf("Generating %s's client from %s to %s...", appName, swaggerURL, relPath))
	basePath := ctx.Path(relPath)
	if err := os.MkdirAll(basePath, os.ModePerm); err != nil {
		return err
	}
	swaggerFile := filepath.Join(basePath, "swagger.json")
	if err := ctx.RunCmd(Cmd{Command: fmt.Sprintf("curl -s %s/api/swagger.json | jq -S . > %s", swaggerURL, swaggerFile)}); err != nil {
		return err
	}
	return ctx.RunCmd(Cmd{Command: fmt.Sprintf(
		"go run %s generate client -f %s --skip-validation -t %s -A %s",
		ctx.VendorPath("github.com/go-swagger/go-swagger/cmd/swagger"), swaggerFile, basePath, appName),
		Env: ctx.CurrentPlatform().GoEnv()})
}

// ProjectRoot finds git root traversing from the current directory up to the dir with .git dir
func (ctx *Context) ProjectRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	_, err = os.Stat(filepath.Join(cwd, "build.yaml"))
	for os.IsNotExist(err) && filepath.Dir(cwd) != "/" {
		cwd = filepath.Dir(cwd)
		_, err = os.Stat(filepath.Join(cwd, "build.yaml"))
	}
	if filepath.Dir(cwd) == "/" {
		panic("Could not determine project root! Make sure to run build from the root!")
	}
	return cwd
}

// GitRoot finds git root traversing from the current directory up to the dir with .git dir
func (ctx *Context) GitRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	_, err = os.Stat(filepath.Join(cwd, ".git"))
	for os.IsNotExist(err) && filepath.Dir(cwd) != "/" {
		cwd = filepath.Dir(cwd)
		_, err = os.Stat(filepath.Join(cwd, ".git"))
	}
	if filepath.Dir(cwd) == "/" {
		panic("Could not determine Git root for the project")
	}
	return cwd
}

// Target returns target by its name defined in config
func (ctx *Context) Target(name string) Target {
	for _, target := range ctx.Targets {
		if target.Name == name {
			return target
		}
	}
	panic("Failed to find target with name: " + name)
}

// Tarball builds tarball for target and platform
func (ctx *Context) BuildBundle(tarball Bundle) error {
	if err := ctx.RunCmd(Cmd{Command: "tar -C " + ctx.OutFileDir(tarball.Target, tarball.Platform) +
		" -czf " + tarball.BinaryFile + " " + ctx.OutFileName(tarball.Target, tarball.Platform)}); err != nil {
		return err
	}

	if err := writeFileChecksum(tarball.BinaryFile, tarball.ChecksumFile); err != nil {
		return err
	}

	return nil
}

// Tarball returns resulting tarball file names for target and platform
func (ctx *Context) Bundle(target Target, platform Platform) Bundle {
	tarName := ctx.BundleFile(target, platform)
	checksumName := fmt.Sprintf("%s.sha256", tarName)
	return Bundle{Target: target, Platform: platform, BinaryFile: tarName, ChecksumFile: checksumName}
}

// BundleFile built file path for target
func (ctx *Context) BundleFile(target Target, platform Platform) string {
	return filepath.Join(ctx.OutFileDir(target, platform), fmt.Sprintf("%s-%s-%s.tar.gz", target.Name, platform.GOOS, platform.GOARCH))
}

// OutFile built file path for target
func (ctx *Context) OutFile(target Target, platform Platform) string {
	return filepath.Join(ctx.OutFileDir(target, platform), ctx.OutFileName(target, platform))
}

func (ctx *Context) OutFileDir(target Target, platform Platform) string {
	return filepath.Join(ctx.OutPath(), fmt.Sprintf("%s-%s", platform.GOOS, platform.GOARCH))
}

// OutFileName file name based for target based on platform
func (ctx *Context) OutFileName(target Target, platform Platform) string {
	var osExt string
	if platform.isWindows() {
		osExt = ".exe"
	}

	return fmt.Sprintf("%s%s", target.Name, osExt)
}

// OutPath returns full path to output directory
func (ctx *Context) OutPath() string {
	return filepath.Join(ctx.ProjectRoot(), ctx.OutDir)
}

// SourceFile returns path to source file for the target
func (ctx *Context) SourceFile(target Target) string {
	return filepath.Join(ctx.ProjectRoot(), target.Path)
}

// ManifestFile returns path to manifest file for the target
func (ctx *Context) ManifestFile(target Target) string {
	return filepath.Join(ctx.ProjectRoot(), filepath.Dir(target.Path), "manifest.toml")
}

// Path returns full path from project root
func (ctx *Context) Path(relPath string) string {
	return filepath.Join(ctx.ProjectRoot(), relPath)
}

// VendorPath returns full path to a file/dir in /vendor dir
func (ctx *Context) VendorPath(relPath string) string {
	return filepath.Join(ctx.ProjectRoot(), "vendor", relPath)
}

// GoEnv returns environment variables necessary for Go to run properly
func (platform Platform) GoEnv() []string {
	res := make([]string, len(GoBuildEnv)+2)
	for _, env := range GoBuildEnv {
		res = append(res, env)
	}
	res = append(res, "GOOS="+platform.GOOS)
	res = append(res, "GOARCH="+platform.GOARCH)
	return res
}

// ListAllSubDirs returns list of all subdirectories under the directory
func (ctx Context) ListAllSubDirs(dir string) ([]string, error) {
	var paths []string
	if err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() && path != dir {
				relPath, err := filepath.Rel(dir, path)
				if err != nil {
					return err
				}
				paths = append(paths, relPath)
			}
			return nil
		}); err != nil {
		return paths, err
	}
	return paths, nil
}

func (platform *Platform) isWindows() bool {
	return platform.GOOS == "windows"
}

func (ctx Context) PlatformsToVariants() []string {
	res := make([]string, len(ctx.Platforms))
	for i, p := range ctx.Platforms {
		res[i] = p.String()
	}
	return res
}

func (p *Platform) String() string {
	return fmt.Sprintf("%s-%s", p.GOOS, p.GOARCH)
}

func writeFileChecksum(inputFile string, outputFile string) error {
	f, err := os.Open(inputFile)

	if err != nil {
		return err
	}

	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return err
	}

	fHash, err := os.Create(outputFile)

	if err != nil {
		return err
	}

	defer fHash.Close()

	_, err = fHash.Write([]byte(fmt.Sprintf("%x", hasher.Sum(nil))))

	return err
}
