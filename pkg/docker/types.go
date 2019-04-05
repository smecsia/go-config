package docker

import (
	"github.com/docker/docker/client"
	"io"
)

// TagDigest represents tag pushed to repository
type TagDigest struct {
	Tag    string
	Digest string
	Size   int
}

// Dockerfile represents the source for Docker image building
type Dockerfile struct {
	FilePath   string
	Tags       []string
	Args       map[string]*string
	ID         string
	TagDigests map[string]TagDigest
	dockerAPI  *client.Client
}

type VolumeMode string

// Volume defines volume to attach to the container
type Volume struct {
	HostPath string
	ContPath string
	Mode     VolumeMode
}

func (m VolumeMode) IsRW() bool {
	return m == "rw"
}

// RunContext defines Docker run configuration
type RunContext struct {
	Prefix         string
	User           string
	Privileged     bool
	DockerInDocker bool

	Stdout  io.WriteCloser
	Stderr  io.WriteCloser
	Stdin   io.ReadCloser
	WorkDir string

	Silent bool
	Debug  bool
	Tty    bool
}

func (ctx *RunContext) isDebug() bool {
	return !ctx.Silent && ctx.Debug
}

// SubWithUser returns copy of context
func (ctx RunContext) Clone() RunContext {
	return RunContext{
		User:           ctx.User,
		Privileged:     ctx.Privileged,
		DockerInDocker: ctx.DockerInDocker,
		Stdout:         ctx.Stdout,
		Stderr:         ctx.Stderr,
		Prefix:         ctx.Prefix,
		WorkDir:        ctx.WorkDir,
		Silent:         ctx.Silent,
		Debug:          ctx.Debug,
	}
}

// Run defines Docker run command
type Run struct {
	RunID       string
	Reference   string
	Volumes     []Volume
	Env         []string
	dockerAPI   *client.Client
	containerID string
}

// ResponseMessage reflects typical response message from Docker daemon
type ResponseMessage struct {
	Id     string `json:"id"`
	Status string `json:"status"`
	Stream string `json:"stream"`
	Aux    struct {
		ID     string `json:"ID"`
		Tag    string `json:"Tag"`
		Digest string `json:"Digest"`
		Size   int    `json:"Size"`
	} `json:"aux"`
	ErrorDetail struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	ProgressDetail struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	}
	Progress string `json:"progress"`
	Error    string `json:"error"`

	summary string `json:"-"`
}

func (m *ResponseMessage) Summary() string {
	return m.summary
}
