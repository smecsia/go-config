package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// NewDockerfile creates new Dockerfile instance with the default client
func NewDockerfile(filePath string, tags ...string) (*Dockerfile, error) {
	dockerAPI, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	dockerAPI.NegotiateAPIVersion(context.Background())
	return &Dockerfile{
		FilePath:   filePath,
		Tags:       tags,
		dockerAPI:  dockerAPI,
		TagDigests: make(map[string]TagDigest),
	}, nil
}

// Validates Dockerfile
// Limitations: very simple validation atm
func (dockerFile *Dockerfile) IsValid() (bool, error) {
	bytes, err := ioutil.ReadFile(dockerFile.FilePath)
	if err != nil {
		return false, errors.Wrapf(err, "failed to read Dockerfile from %s", dockerFile.FilePath)
	}
	return strings.Contains(string(bytes), "FROM "), nil
}

// Build builds Docker image from the specified Dockerfile with the specified tags
func (dockerFile *Dockerfile) Build() (MsgReader, error) {

	msgChan := make(chan readerNextMessage)
	msgReader := chanMsgReader{msgChan: msgChan, expectedEOFs: 1}

	buildOptions := types.ImageBuildOptions{
		SuppressOutput: false,
		PullParent:     true,
		Tags:           dockerFile.Tags,
		NoCache:        true,
		Dockerfile:     filepath.Base(dockerFile.FilePath),
		BuildArgs:      dockerFile.Args,
		//Version:        types.BuilderBuildKit, // important to run on V2 build engine to enable buildkit extensions
	}

	resp, err := dockerFile.dockerAPI.ImageBuild(context.Background(), dockerFile.getTarWithOptions(), buildOptions)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(resp.Body)

	go dockerFile.streamMessagesToChannel(reader, msgChan, "")

	return &msgReader, nil
}

// Push pushes Docker image to the registry defined by its tag
func (dockerFile *Dockerfile) Push() (MsgReader, error) {
	if len(dockerFile.Tags) == 0 {
		return nil, errors.New("no tags provided, hence could not push image")
	}

	msgChan := make(chan readerNextMessage)
	msgReader := chanMsgReader{msgChan: msgChan, expectedEOFs: len(dockerFile.Tags)}

	for _, tag := range dockerFile.Tags {
		registry, err := RegistryFromImageReference(tag)
		if err != nil {
			return nil, err
		}
		pushOpts := types.ImagePushOptions{
			RegistryAuth: registry.AuthHeader,
		}
		resp, err := dockerFile.dockerAPI.ImagePush(context.Background(), tag, pushOpts)
		if err != nil {
			return nil, err
		}
		reader := bufio.NewReader(resp)

		go dockerFile.streamMessagesToChannel(reader, msgChan, tag)
	}
	return &msgReader, nil
}

func (dockerFile *Dockerfile) streamMessagesToChannel(reader *bufio.Reader, msgChan chan readerNextMessage, fullTag string) {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				msgChan <- readerNextMessage{EOF: true}
				break
			} else {
				msgChan <- readerNextMessage{Error: err}
			}
		}
		//fmt.Println(line)
		msg := ResponseMessage{}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			msgChan <- readerNextMessage{Error: err}
		} else {
			if msg.Aux.ID != "" {
				dockerFile.ID = msg.Aux.ID
			} else if msg.Aux.Digest != "" && msg.Aux.Tag != "" {
				tagDigest := TagDigest{
					Tag:    msg.Aux.Tag,
					Digest: msg.Aux.Digest,
					Size:   msg.Aux.Size,
				}
				dockerFile.TagDigests[fullTag] = tagDigest
			} else if msg.Error != "" {
				msgChan <- readerNextMessage{Error: errors.New(msg.Error)}
			} else {
				msgChan <- readerNextMessage{Message: msg}
			}
		}
	}
}

func (dockerFile *Dockerfile) getTarWithOptions() io.Reader {
	ctx, _ := archive.TarWithOptions(filepath.Dir(dockerFile.FilePath), &archive.TarOptions{})
	return ctx
}
