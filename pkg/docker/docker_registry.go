package docker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"strings"
)

type Image struct {
	Reference string
	Name      string
	Digest    string
}

type Registry struct {
	AuthHeader string
}

// ResolveDockerImageReference resolves valid docker image reference
// reference can be represented in the format <image-name>@<digest> or <image-name>:<tag>
func ResolveDockerImageReference(reference string) (Image, error) {
	ref, err := name.ParseReference(reference, name.WeakValidation)
	if err != nil {
		return Image{}, fmt.Errorf("parsing reference %q: %v", reference, err)
	}
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return Image{}, fmt.Errorf("reading image %q: %v", ref, err)
	}
	hash, err := img.Digest()
	if err != nil {
		return Image{}, fmt.Errorf("gettig image hash %q: %v", ref, err)
	}
	return Image{
		Reference: fmt.Sprintf("%s@%s", ref.Context(), hash),
		Name:      ref.Context().String(),
		Digest:    hash.String(),
	}, nil
}

// ImageFromReference returns image name from full reference
func ImageFromReference(reference string) (string, error) {
	ref, err := name.ParseReference(reference, name.WeakValidation)
	if err != nil {
		return "", fmt.Errorf("parsing reference %q: %v", reference, err)
	}
	return ref.Context().String(), nil
}

// RegistryFromImageReference allows to figure out the Docker registry context, such as authentication
// this can be useful when working with Docker registry through Docker daemon and SDK
// Limitations: only certain auth methods are supported (e.g. Basic)
func RegistryFromImageReference(reference string) (*Registry, error) {
	ref, err := name.ParseReference(reference, name.WeakValidation)
	if err != nil {
		return nil, fmt.Errorf("parsing reference %q: %v", reference, err)
	}
	creds, err := authn.DefaultKeychain.Resolve(ref.Context().Registry)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid creds for registry %q: %v", ref.Context().RegistryStr(), err)
	}
	var authHeader string
	authHeader, err = creds.Authorization()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth from creds for registry %q: %v", ref.Context().RegistryStr(), err)
	}
	authConfig := types.AuthConfig{}
	if strings.HasPrefix(authHeader, "Basic ") {
		authHeader = strings.Replace(authHeader, "Basic ", "", 1)
		userPassBytes, err := base64.URLEncoding.DecodeString(authHeader)
		if err != nil {
			return nil, fmt.Errorf("failed to decode Basic auth header values for repo %q: %v", ref.Context().RegistryStr(), err)
		}
		userPass := strings.Split(string(userPassBytes), ":")
		authConfig.Username = userPass[0]
		authConfig.Password = userPass[1]
	} else {
		authConfig.RegistryToken = authHeader
	}
	if token, err := encodeDockerAuthHeader(authConfig); err != nil {
		return nil, err
	} else {
		return &Registry{AuthHeader: token}, nil
	}
}

func encodeDockerAuthHeader(authConfig types.AuthConfig) (string, error) {
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(encodedJSON), nil
}
