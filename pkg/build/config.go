package build

import (
	"github.com/smecsia/go-utils/pkg/config"
	"github.com/smecsia/go-utils/pkg/util"
)

// Writes config file to yaml file
func WriteConfigFile(filePath string, cfg *Context) error {
	return config.WriteConfigFile(filePath, cfg)
}

// DefaultConfig Returns default version of Config file
func DefaultConfig() *Context {
	return config.DefaultConfig(&Context{}).(*Context)
}

// Reads config file from yaml safely and adds defaults from env or default tags
func Init(filePath string, reader util.ConsoleReader) *Context {
	return config.Init(filePath, &Context{}, reader).(*Context)
}

// ReadConfigFile Reads config file from yaml file
func ReadConfigFile(filePath string) (*Context, map[string]interface{}, error) {
	ctx, raw, err := config.ReadConfigFile(filePath, &Context{})
	return ctx.(*Context), raw, err
}
