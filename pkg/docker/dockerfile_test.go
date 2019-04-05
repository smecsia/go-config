package docker_test

import (
	. "github.com/smecsia/go-utils/pkg/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestValidDockerfile_Build(t *testing.T) {
	dockerFile := newDockerFile(t, "testdata/ValidDockerfile")
	requiredVal := "required_val"
	optionalVal := "optional_val"
	dockerFile.Args = map[string]*string{
		"required_arg": &requiredVal,
		"optional_arg": &optionalVal,
	}
	reader, err := dockerFile.Build()

	require.NoError(t, err)
	err = reader.Listen(true, func(message *ResponseMessage, err error) {
		require.NoError(t, err)
	})
	require.NoError(t, err)
	assert.NotEmpty(t, dockerFile.ID)
}

func TestInvalidRunDockerfile_Build(t *testing.T) {
	dockerFile := newDockerFile(t, "testdata/InvalidRunDockerfile")

	reader, err := dockerFile.Build()

	require.NoError(t, err)

	var expectErr error
	for {
		res, err := reader.Next()
		if err != nil {
			assert.Contains(t, err.Error(), "some-invalid-command", "returned a non-zero code")
			expectErr = err
		} else if res == nil {
			break
		}
	}
	require.Error(t, expectErr)
}

func TestInvalidDockerfile_Build(t *testing.T) {
	dockerFile := newDockerFile(t, "testdata/InvalidDockerfile")

	_, err := dockerFile.Build()

	require.Error(t, err)
}

func newDockerFile(t *testing.T, path string) *Dockerfile {
	path, err := filepath.Abs(path)
	require.NoError(t, err)
	dockerFile, err := NewDockerfile(path, "docker.atl-paas.net/deng/test/ok:blah1", "docker.atl-paas.net/deng/test/ok:blah2")
	require.NoError(t, err)
	return dockerFile
}
