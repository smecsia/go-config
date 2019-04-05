package build

import (
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
)

// SetVersionInConfig sets new version in the config file
func (ctx *Context) SetVersionInConfig(version string) error {
	cfg, _, err := ReadConfigFile(ctx.GetConfigFilePath())
	if err != nil {
		return err
	}
	cfg.Version = version
	return WriteConfigFile(ctx.GetConfigFilePath(), cfg)
}

// SemVer returns parsed SemVer object
func (ctx *Context) SemVer() *semver.Version {
	if v, err := semver.NewVersion(ctx.Version); err != nil {
		panic(err)
	} else {
		return v
	}
}

// FullVersion returns version + git hash
func (ctx *Context) FullVersion() string {
	hash, err := ctx.git.HashShort()
	if err != nil {
		panic(errors.Wrap(err, "unable to detect Git hash"))
	}
	return fmt.Sprintf("%s-%s", ctx.Version, hash)
}
