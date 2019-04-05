package config_test

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	. "github.com/smecsia/go-utils/pkg/config"
)

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

// Context build context and config
type TestConfig struct {
	OutDir       string     `yaml:"outDir,omitempty" env:"OUT_DIR" default:"bin"`
	Version      string     `yaml:"version,omitempty" env:"VERSION" default:""`
	ArmoryURL    string     `yaml:"armoryURL,omitempty" env:"ARMORY_URL" default:"https://armory.prod.atl-paas.net"`
	TrebuchetURL string     `yaml:"trebuchetURL,omitempty" env:"TREBUCHET_URL" default:"https://trebuchet.prod.atl-paas.net"`
	Platforms    []Platform `yaml:"platforms,omitempty"`
	Targets      []Target   `yaml:"targets,omitempty"`

	// env-only fields
	IsParallel  bool   `yaml:"-" default:"true" env:"PARALLEL"`
	IsSkipTests string `yaml:"-" default:"false" env:"SKIP_TESTS"`

	// default-only fields
	configFilePath string
	initialized    bool
}

func (tc *TestConfig) SetConfigFilePath(path string) {
	tc.configFilePath = path
}

func (tc *TestConfig) GetConfigFilePath() string {
	return tc.configFilePath
}

func (tc *TestConfig) Init() error {
	tc.initialized = true
	return nil
}

func TestNewConfig(t *testing.T) {
	RegisterTestingT(t)

	defaultConfig := DefaultConfig(&TestConfig{}).(*TestConfig)
	result := AddDefaults(map[string]interface{}{"armoryURL": "", "outDir": ""}, &TestConfig{ArmoryURL: "http://blabla", OutDir: "bla"}).(*TestConfig)
	Expect(result.initialized).To(Equal(false))
	Expect(result.ArmoryURL).To(Equal("http://blabla"))
	Expect(result.OutDir).To(Equal("bla"))
	Expect(result.TrebuchetURL).To(Equal(defaultConfig.TrebuchetURL))
}

func TestNewConfigWithEnvironment(t *testing.T) {
	RegisterTestingT(t)

	defer os.Setenv("OUT_DIR", "")
	defer os.Setenv("ARMORY_URL", "")
	defer os.Setenv("PARALLEL", "")
	os.Setenv("ARMORY_URL", "http://blablabla")
	os.Setenv("OUT_DIR", "somedir")
	os.Setenv("PARALLEL", "false")

	config := AddEnv(DefaultConfig(&TestConfig{})).(*TestConfig)
	Expect(config.initialized).To(Equal(false))
	Expect(config.ArmoryURL).To(Equal("http://blablabla"))
	Expect(config.OutDir).To(Equal("somedir"))
	Expect(config.IsParallel).To(Equal(false))
}

type MockedReader struct {
	mock.Mock
}

func (m *MockedReader) ReadPassword() (string, error) {
	args := m.Called()
	return args.String(0), nil
}

func (m *MockedReader) ReadLine() (string, error) {
	args := m.Called()
	return args.String(0), nil
}

func TestReadConfig(t *testing.T) {
	RegisterTestingT(t)
	mockedReader := new(MockedReader)
	defaultConfig := DefaultConfig(&TestConfig{}).(*TestConfig)
	mockedReader.On("ReadLine").Return("1.0.0")
	defer os.Setenv("ARMORY_URL", "")
	defer os.Setenv("SKIP_TESTS", "")
	os.Setenv("ARMORY_URL", "http://blablabla")
	os.Setenv("SKIP_TESTS", "true")

	config := ReadConfig(AddEnv(DefaultConfig(&TestConfig{})), mockedReader).(*TestConfig)
	Expect(config.initialized).To(Equal(false))
	Expect(config.ArmoryURL).To(Equal("http://blablabla"))
	Expect(config.Version).To(Equal("1.0.0"))
	Expect(config.TrebuchetURL).To(Equal(defaultConfig.TrebuchetURL))
	Expect(config.IsParallel).To(Equal(true))
	Expect(config.IsSkipTests).To(Equal("true"))
}

func TestItShouldNotReadVersionIfSetInEnvVar(t *testing.T) {
	RegisterTestingT(t)
	mockedReader := new(MockedReader)
	mockedReader.On("ReadLine").Return("SomeString")
	defer os.Setenv("VERSION", "")
	os.Setenv("VERSION", "1.0.0")

	config := ReadConfig(AddEnv(DefaultConfig(&TestConfig{})), mockedReader).(*TestConfig)

	Expect(config.Version).To(Equal("1.0.0"))
	mockedReader.AssertNotCalled(t, "ReadLine")
}

func TestReadConfigFile(t *testing.T) {
	RegisterTestingT(t)

	readConfig, rawConfig, err := ReadConfigFile("testdata/build.yaml", AddEnv(DefaultConfig(&TestConfig{})))
	Expect(err).To(BeNil())

	config := AddDefaults(rawConfig, readConfig).(*TestConfig)

	defaultConfig := DefaultConfig(&TestConfig{}).(*TestConfig)
	Expect(config.initialized).To(Equal(false))
	Expect(config.ArmoryURL).To(Equal("http://armory.local"))
	Expect(config.TrebuchetURL).To(Equal(defaultConfig.TrebuchetURL))
	Expect(config.Platforms).To(HaveLen(2))
	Expect(config.Targets).To(HaveLen(1))
	Expect(config.IsParallel).To(Equal(true))
	Expect(config.GetConfigFilePath()).To(Equal("testdata/build.yaml"))
	Expect(config.IsSkipTests).To(Equal("false"))
}

func TestInit(t *testing.T) {
	RegisterTestingT(t)

	defer os.Setenv("SKIP_TESTS", "")
	os.Setenv("SKIP_TESTS", "true")
	mockedReader := new(MockedReader)
	mockedReader.On("ReadLine").Return("1.0.0")
	config := Init("testdata/build.yaml", &TestConfig{}, mockedReader).(*TestConfig)

	defaultConfig := DefaultConfig(&TestConfig{}).(*TestConfig)
	Expect(config.initialized).To(Equal(true))
	Expect(config.ArmoryURL).To(Equal("http://armory.local"))
	Expect(config.TrebuchetURL).To(Equal(defaultConfig.TrebuchetURL))
	Expect(config.Platforms).To(HaveLen(2))
	Expect(config.Targets).To(HaveLen(1))
	Expect(config.IsParallel).To(Equal(true))
	Expect(config.GetConfigFilePath()).To(Equal("testdata/build.yaml"))
	Expect(config.IsSkipTests).To(Equal("true"))
}
