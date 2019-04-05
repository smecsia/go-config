package docker_test

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/smecsia/go-utils/pkg/docker"
)

func createTempDir(prefix string) (string, error) {
	if err := os.MkdirAll("testdata/tmp", os.ModePerm); err != nil {
		return "", err
	}
	if systemTmp, err := ioutil.TempDir("", prefix); err != nil {
		return "", err
	} else {
		testdataTmpDir, err := filepath.Abs(path.Join("testdata/tmp", path.Base(systemTmp)))
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(testdataTmpDir, os.ModePerm); err != nil {
			return "", err
		}
		return testdataTmpDir, nil
	}
}

func TestDockerRun(t *testing.T) {
	RegisterTestingT(t)

	cwd, err := os.Getwd()
	Expect(err).To(BeNil())

	tmpDir, err := createTempDir("tmpdir")
	tmpDir2, err := createTempDir("tmpdir2")
	defer os.RemoveAll(tmpDir)
	defer os.RemoveAll(tmpDir2)

	Expect(err).To(BeNil())

	dockerRun, err := NewRun("test123", "docker:latest",
		Volume{
			HostPath: path.Join(cwd, "testdata"),
			ContPath: "/test",
		},
		Volume{
			HostPath: path.Join(cwd, "testdata", "ValidDockerfile"),
			ContPath: "/some/valid/dockerfile/path",
		},
		Volume{
			HostPath: tmpDir,
			ContPath: "/tmp/tempdir",
			Mode:     "rw",
		},
		Volume{
			HostPath: tmpDir2,
			ContPath: "/tmp/tempdir2",
			Mode:     "rw",
		},
	)

	Expect(err).To(BeNil())

	curUser, err := user.Current()

	Expect(err).To(BeNil())

	reader, stdout := io.Pipe()

	var eg errgroup.Group

	eg.Go(func() error {
		var buffer bytes.Buffer
		bufReader := bufio.NewReader(reader)
		for {
			raw, _, err := bufReader.ReadLine()
			buffer.Write(raw)
			switch err {
			case io.EOF:
				output := string(buffer.String())
				Expect(output).To(ContainSubstring("[docker / test123] > " + curUser.Username))
				Expect(output).To(ContainSubstring("[docker / test123] > OK"))
				Expect(output).To(MatchRegexp(fmt.Sprintf("\\[docker \\/ test123\\] > [rw\\-]+\\s+1\\s+%s\\s+%s\\s+\\d+\\s+\\w+\\s+\\d+\\s+\\d+:\\d+\\s+InvalidDockerfile", curUser.Username, curUser.Username)))
				Expect(output).To(MatchRegexp(fmt.Sprintf("\\[docker \\/ test123\\] > [rw\\-]+\\s+1\\s+%s\\s+%s\\s+\\d+\\s+\\w+\\s+\\d+\\s+\\d+:\\d+\\s+path", curUser.Username, curUser.Username)))

				outFilePath := path.Join(tmpDir, "somefile")
				_, err = os.Stat(outFilePath)
				Expect(os.IsNotExist(err)).To(Equal(false), "file "+outFilePath+" must exist")

				outFile2Path := path.Join(tmpDir2, "output", "somefile")
				_, err = os.Stat(outFile2Path)
				Expect(os.IsNotExist(err)).To(Equal(false), "file "+outFile2Path+" must exist")

				outFile2OutputPath := path.Join(tmpDir2, "output", "output", "somefile")
				_, err = os.Stat(outFile2OutputPath)
				Expect(os.IsNotExist(err)).To(Equal(true), "file "+outFile2OutputPath+" must not exist")

				return nil
			case nil:
				fmt.Println(string(raw))
			default:
				return errors.Wrapf(err, "failed to read next line from build output")
			}
		}
	})

	var out io.WriteCloser = stdout
	err = dockerRun.Run(RunContext{Prefix: " \t [docker / test123] > ", Stdout: out, DockerInDocker: true, User: curUser.Username, Debug: true},
		"whoami", "sh -c 'docker images && sleep 0.1 && ps -a && sleep 0.1 && ls -la /test'",
		"sh -c 'ls -la /some/valid/dockerfile && echo OK'",
		"echo 'BLAH' > /tmp/tempdir/somefile",
		"mkdir /tmp/tempdir2/output && echo 'BLAH2' > /tmp/tempdir2/output/somefile",
	)

	Expect(err).To(BeNil())
	Expect(eg.Wait()).To(BeNil())
}
