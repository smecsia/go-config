package docker_test

import (
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	. "github.com/smecsia/go-utils/pkg/docker"
	"testing"
)

func TestDockerImageFromReference(t *testing.T) {
	RegisterTestingT(t)

	img, err := ImageFromReference("docker.atl-paas.net/deng/test/ok:blah1")
	Expect(err).To(BeNil())

	Expect(img).To(Equal("docker.atl-paas.net/deng/test/ok"))
}

func TestResolveImageWithTag(t *testing.T) {
	image, err := ResolveDockerImageReference("ubuntu")

	require.NoError(t, err)

	assert.Equal(t, "index.docker.io/library/ubuntu", image.Name)
	assert.Regexp(t, "ubuntu@sha256:[0-9A-Fa-f]{32,}", image.Reference)
	assert.Regexp(t, "sha256:[0-9A-Fa-f]{32,}", image.Digest)
}

func TestResolveImageWithDigest(t *testing.T) {
	image, err := ResolveDockerImageReference("ubuntu@sha256:de774a3145f7ca4f0bd144c7d4ffb2931e06634f11529653b23eba85aef8e378")

	require.NoError(t, err)

	assert.Equal(t, "index.docker.io/library/ubuntu", image.Name)
	assert.Regexp(t, "ubuntu@sha256:[0-9A-Fa-f]{32,}", image.Reference)
	assert.Regexp(t, "sha256:[0-9A-Fa-f]{32,}", image.Digest)
}

func TestResolveNotExistingImage(t *testing.T) {
	_, err := ResolveDockerImageReference("not-existing-user/some-not-existing-image1234567890")

	require.Error(t, err)
}
