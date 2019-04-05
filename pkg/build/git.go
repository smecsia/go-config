package build

import "fmt"

// IsWorkTreeClean checks if a worktree is clean (no new changes have been produced)
func (ctx *Context) GitCheckWorkTree() error {
	if clean, status, err := ctx.git.IsWorkTreeClean(); err != nil {
		return err
	} else if !clean {
		return fmt.Errorf("git tree is not clean: \n%s", status)
	}
	return nil
}

// CommitAndPush makes commit and pushes to master
func (ctx *Context) GitCommitAndPush(msg string) error {
	return ctx.git.CommitAndPush(msg)
}
