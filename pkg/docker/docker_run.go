package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/system"
	"github.com/pkg/errors"
	"github.com/segmentio/textio"
	"github.com/smecsia/go-utils/pkg/docker/dockerext"
	"github.com/smecsia/go-utils/pkg/util"
	"golang.org/x/sync/errgroup"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	ContainersLabelName      = "AtlasBuildContainerID"
	DefaultContainerCommand  = "sleep 100000"
	ValidUsernameRegexString = `^[a-z_]([a-z0-9_-]{0,31}|[a-z0-9_-]{0,30}\$)$`
	DockerSockPath           = "/var/run/docker.sock"
)

var ValidUsernameRegex = regexp.MustCompile(ValidUsernameRegexString)

// NewRun creates new Run instance with the default client
func NewRun(runID string, ref string, volumes ...Volume) (*Run, error) {
	dockerAPI, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	dockerAPI.NegotiateAPIVersion(context.Background())
	return &Run{
		RunID:     runID,
		Volumes:   volumes,
		Reference: ref,
		dockerAPI: dockerAPI,
	}, nil
}

// IsRunningInDocker returns true if currently running inside Docker container
func (run *Run) IsRunningInDocker() bool {
	//fmt.Println("check if running in Docker...")
	if bytes, err := ioutil.ReadFile("/proc/1/cgroup"); err != nil {
		//fmt.Println("read /proc/1/cgroup unsuccessful: ", err)
		return false
	} else {
		//fmt.Println("read /proc/1/cgroup successful, result: ", string(bytes))
		return strings.Contains(string(bytes), "docker") ||
			strings.Contains(string(bytes), "kubepods")
	}
}

// Run executes commands
func (run *Run) Run(runCtx RunContext, commands ...string) error {
	if err := run.Destroy(); err != nil {
		return err
	}

	defer run.Destroy()
	if runCtx.Stdout != os.Stdout && runCtx.Stdout != nil {
		defer runCtx.Stdout.Close()
	}

	if runCtx.Stderr != os.Stderr && runCtx.Stderr != nil {
		defer runCtx.Stderr.Close()
	}

	if err := run.makeSureImagePulled(runCtx); err != nil {
		return err
	}

	var binds []string
	if runCtx.DockerInDocker {
		binds = append(binds, fmt.Sprintf("%s:%s", DockerSockPath, DockerSockPath))
	}

	// make binds out of volumes if not running in container
	if !run.IsRunningInDocker() {
		for _, v := range run.Volumes {
			bind := fmt.Sprintf("%s:%s", v.HostPath, v.ContPath)
			if v.Mode == "ro" {
				bind += ":ro"
			}
			binds = append(binds, bind)
		}
	}

	config := &container.Config{
		Image: run.Reference,
		Labels: map[string]string{
			ContainersLabelName: run.RunID,
		},
		Cmd:        util.SafeSplit(DefaultContainerCommand),
		Entrypoint: []string{},
		Env:        run.Env,
	}
	hostConfig := &container.HostConfig{
		NetworkMode: "default",
		Privileged:  runCtx.Privileged,
		Binds:       binds,
	}
	networkConfig := &network.NetworkingConfig{}
	ctx := context.Background()

	createResp, err := run.dockerAPI.ContainerCreate(ctx, config, hostConfig, networkConfig, run.RunID)
	if err != nil {
		return err
	}

	run.containerID = createResp.ID

	if err := run.dockerAPI.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	if err := run.makeSureUserExists(runCtx, run.containerID); err != nil {
		return err
	}

	// copy volumes contents into container if running in dind
	if run.IsRunningInDocker() {
		if err := run.copyVolumesToContainer(runCtx); err != nil {
			return err
		}
	}

	// run all commands inside container
	for _, cmd := range commands {
		if err := run.execSingleCommand(runCtx, run.containerID, cmd, false); err != nil {
			return err
		}
	}

	// copy rw volumes contents from container if running in dind
	if run.IsRunningInDocker() {
		if err := run.copyVolumesFromContainer(runCtx); err != nil {
			return err
		}
	}

	return nil
}

func (run *Run) copyVolumesFromContainer(runCtx RunContext) error {
	if runCtx.isDebug() {
		_, _ = runCtx.Stdout.Write([]byte(fmt.Sprintf("Docker In Docker mode activated: copying volumes from container!\n")))
	}
	// get all changed paths since container started (will return all sub directories / files of all volumes)
	diffPaths, err := run.dockerAPI.ContainerDiff(context.Background(), run.containerID)
	if err != nil {
		return errors.Wrapf(err, "failed to get diff for container %s", run.containerID)
	}
	for _, v := range run.Volumes {
		// copying only RW volumes back to host
		if v.Mode.IsRW() {
			for _, diffPath := range diffPaths {
				// check whether change is a part of volume or the volume itself (in case of file)
				partOfVolume := strings.HasPrefix(diffPath.Path, path.Clean(v.ContPath)+"/") || diffPath.Path == path.Clean(v.ContPath)
				if partOfVolume {
					contPath := diffPath.Path
					relPath, err := filepath.Rel(path.Clean(v.ContPath), contPath)
					if err != nil {
						return errors.Wrapf(err, "failed to calculate relative path of changed container mount %s off %s", diffPath.Path, v.ContPath)
					}
					// skip if file/dir is not a direct child of a volume
					if strings.Contains(relPath, "/") || contPath == v.ContPath {
						continue
					}
					if err := run.copyFromContainer(runCtx, contPath, v.HostPath); err != nil {
						return errors.Wrapf(err, "could not copy file %s of volume %s:%s back from container", relPath, v.HostPath, v.ContPath)
					}
				}
			}
		}
	}
	return nil
}

func (run *Run) copyVolumesToContainer(runCtx RunContext) error {
	if runCtx.isDebug() {
		_, _ = runCtx.Stdout.Write([]byte(fmt.Sprintf("Docker In Docker mode activated: copying volumes into container!\n")))
	}
	for _, v := range run.Volumes {
		if err := run.copyToContainer(runCtx, v.HostPath, v.ContPath); err != nil {
			return errors.Wrapf(err, "could not copy volume %s:%s to container %s", v.HostPath, v.ContPath, run.containerID)
		}
	}
	return nil
}

func (run *Run) copyFromContainer(runCtx RunContext, contPath string, hostPath string) error {
	if runCtx.isDebug() {
		_, _ = runCtx.Stdout.Write([]byte(fmt.Sprintf("COPY %s:%s %s\n", run.containerID, contPath, hostPath)))
	}
	// if client requests to follow symbol link, then must decide target file to be copied
	var rebaseName string
	contStat, err := run.dockerAPI.ContainerStatPath(context.Background(), run.containerID, contPath)

	// If the source is a symbolic link, we should follow it.
	if err == nil && contStat.Mode&os.ModeSymlink != 0 {
		linkTarget := contStat.LinkTarget
		if !system.IsAbs(linkTarget) {
			// Join with the parent directory.
			srcParent, _ := archive.SplitPathDirEntry(contPath)
			linkTarget = filepath.Join(srcParent, linkTarget)
		}

		linkTarget, rebaseName = archive.GetRebaseName(contPath, linkTarget)
		contPath = linkTarget
	}

	content, stat, err := run.dockerAPI.CopyFromContainer(context.Background(), run.containerID, contPath)
	if err != nil {
		return err
	}
	defer content.Close()

	if hostPath == "-" {
		// Send the response to STDOUT.
		_, err = io.Copy(runCtx.Stdout, content)

		return err
	}

	// Prepare source copy info.
	srcInfo := archive.CopyInfo{
		Path:       contPath,
		Exists:     true,
		IsDir:      stat.Mode.IsDir(),
		RebaseName: rebaseName,
	}

	preArchive := content
	if len(srcInfo.RebaseName) != 0 {
		_, srcBase := archive.SplitPathDirEntry(srcInfo.Path)
		preArchive = archive.RebaseArchiveEntries(content, srcBase, srcInfo.RebaseName)
	}
	// See comments in the implementation of `archive.CopyTo` for exactly what
	// goes into deciding how and whether the source archive needs to be
	// altered for the correct copy behavior.
	return archive.CopyTo(preArchive, srcInfo, hostPath)
}

func (run *Run) copyToContainer(runCtx RunContext, hostPath string, contPath string) error {
	if !runCtx.Silent && runCtx.Debug {
		_, _ = runCtx.Stdout.Write([]byte(fmt.Sprintf("COPY %s %s:%s\n", hostPath, run.containerID, contPath)))
	}
	srcInfo, err := os.Stat(hostPath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.Wrapf(err, "path %s does not exist in the host container", hostPath)
		} else if os.IsPermission(err) {
			return errors.Wrapf(err, "path %s is not accessible in the host container", hostPath)
		}
	}
	srcFileName := path.Base(hostPath)
	dstFileName := path.Base(contPath)
	dstDirPath := contPath
	if !srcInfo.IsDir() {
		dstDirPath = path.Dir(contPath)
	}
	rootCtx := runCtx.Clone()
	rootCtx.User = "root"
	rootCtx.WorkDir = "/"

	if dstDirPath != DockerSockPath {
		mkdirCmd := "mkdir -p " + dstDirPath
		if runCtx.User != "" && runCtx.User != "root" {
			mkdirCmd += " && chown " + runCtx.User + ":" + runCtx.User + " " + dstDirPath
		}
		if err := run.execSingleCommand(rootCtx, run.containerID, mkdirCmd, true); err != nil {
			return err
		}
	}

	tar, err := archive.TarWithOptions(hostPath, &archive.TarOptions{})

	if err != nil {
		return errors.Wrapf(err, "failed to create tar archive out of the path: %s", hostPath)
	}

	err = run.dockerAPI.CopyToContainer(context.Background(), run.containerID, dstDirPath, tar, types.CopyToContainerOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to copy files into container %s from path: %s", run.containerID, hostPath)
	}
	if !srcInfo.IsDir() && dstFileName != srcFileName {
		renameCmd := "mv " + dstDirPath + "/" + srcFileName + " " + dstDirPath + "/" + dstFileName
		if err := run.execSingleCommand(rootCtx, run.containerID, renameCmd, true); err != nil {
			return err
		}
	}

	return nil
}

func (run *Run) makeSureUserExists(runCtx RunContext, containerID string) error {
	if runCtx.User == "" || !ValidUsernameRegex.Match([]byte(runCtx.User)) {
		return nil
	}
	rootCtx := runCtx.Clone()
	rootCtx.User = "root"
	rootCtx.WorkDir = "/"
	// add user respecting Alpine's & Debian's command arguments:
	addUserCmd := fmt.Sprintf("sh -c 'id -u %s 2>/dev/null || "+
		"adduser -D %s >/dev/null 2>&1  || "+
		"adduser --disabled-password --gecos \"\" %s >/dev/null 2>&1'", runCtx.User, runCtx.User, runCtx.User)
	if err := run.execSingleCommand(rootCtx, containerID, addUserCmd, true); err != nil {
		return err
	}
	homeDir := fmt.Sprintf("/home/%s", runCtx.User)
	if runCtx.User == "root" {
		homeDir = "/root"
	}
	if runCtx.User != "root" {
		if err := run.execSingleCommand(rootCtx, containerID, fmt.Sprintf("chown -R %s:%s %s", runCtx.User, runCtx.User, homeDir), true); err != nil {
			return err
		}
	}

	// permission for the user to access docker.sock is required to run dind containers
	if runCtx.DockerInDocker {
		return run.execSingleCommand(rootCtx, containerID, fmt.Sprintf("chmod +rx %s", DockerSockPath), true)
	}
	return nil
}

func (run *Run) Destroy() error {
	ctx := context.Background()
	containers, err := run.dockerAPI.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return err
	}
	var eg errgroup.Group
	for _, c := range containers {
		if c.Labels[ContainersLabelName] == run.RunID {
			eg.Go(func() error {
				timeout := time.Second * 10
				_ = run.dockerAPI.ContainerStop(ctx, c.ID, &timeout)
				_ = run.dockerAPI.ContainerKill(ctx, c.ID, "KILL")
				return run.dockerAPI.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{
					RemoveVolumes: true,
					Force:         true,
				})
			})
		}
	}
	return eg.Wait()
}

func (run *Run) makeSureImagePulled(runCtx RunContext) error {

	opts := types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", run.Reference)),
	}
	ctx := context.Background()
	images, err := run.dockerAPI.ImageList(ctx, opts)
	if err != nil {
		return err
	}
	if len(images) == 0 {
		// pull image
		dockerMsgReader := chanMsgReader{msgChan: make(chan readerNextMessage)}

		reader, err := run.dockerAPI.ImagePull(ctx, run.Reference, types.ImagePullOptions{})
		if err != nil {
			return err
		}

		var eg errgroup.Group

		eg.Go(func() error {
			return run.streamMessagesToChannel(bufio.NewReader(reader), dockerMsgReader.msgChan)
		})

		if runCtx.Stdout != nil {
			if err := dockerMsgReader.Listen(false, func(message *ResponseMessage, err error) {
				_, _ = runCtx.Stdout.Write([]byte(runCtx.Prefix + strings.Replace(message.summary, "\n", "\n"+runCtx.Prefix, -1)))
			}); err != nil {
				return err
			}
		}

		return eg.Wait()
	}
	return nil

}

func (run *Run) execSingleCommand(runCtx RunContext, containerID string, command string, svcCmd bool) error {
	command = strings.TrimSpace(command)
	execConfig := types.ExecConfig{
		Privileged:   true,
		Cmd:          []string{"/bin/sh", "-c", command},
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  true,
		Detach:       false,
		Tty:          runCtx.Tty,
	}

	stdout := io.Writer(runCtx.Stdout)
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := io.Writer(runCtx.Stderr)
	if stderr == nil {
		stderr = os.Stderr
	}
	if !runCtx.Tty {
		stderr = textio.NewPrefixWriter(stderr, runCtx.Prefix)
		stdout = textio.NewPrefixWriter(stdout, runCtx.Prefix)
	}

	if !svcCmd && runCtx.isDebug() {
		_, err := stdout.Write([]byte(fmt.Sprintf("CMD `%s`\n", command)))
		if err != nil {
			return err
		}
	}

	if runCtx.WorkDir != "" {
		execConfig.WorkingDir = runCtx.WorkDir
	}

	if runCtx.User != "" && ValidUsernameRegex.Match([]byte(runCtx.User)) {
		execConfig.User = runCtx.User
	}

	ctx := context.Background()
	crResp, err := run.dockerAPI.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return err
	}

	if err := dockerext.InteractiveExec(run.dockerAPI, dockerext.InteractiveExecCfg{
		ExecID:     crResp.ID,
		Context:    context.Background(),
		Stderr:     stderr,
		Stdout:     stdout,
		Stdin:      runCtx.Stdin,
		ExecConfig: &execConfig,
	}); err != nil {
		return err
	}
	return nil
}

func (run *Run) streamMessagesToChannel(reader *bufio.Reader, msgChan chan readerNextMessage) error {
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
		msg := ResponseMessage{}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			msgChan <- readerNextMessage{Error: err}
		} else {
			if msg.Error != "" {
				msgChan <- readerNextMessage{Error: errors.New(msg.Error)}
			} else {
				msgChan <- readerNextMessage{Message: msg}
			}
		}
	}
	return nil
}
