package git

import (
	"github.com/pkg/errors"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"os"
	"path/filepath"
	"time"
)

type Git interface {
	IsWorkTreeClean() (bool, string, error)
	HashShort() (string, error)
	Hash() (string, error)
	CommitAndPush(msg string) error
	Root() string
}

type GitImpl struct {
	RootPath string
	Author   string
	Branch   string
	Remote   string
}

func New(gitRoot string) Git {
	return &GitImpl{
		RootPath: gitRoot,
	}
}

func NewWithCfg(gitRoot string, author string, branch string, remote string) Git {
	return &GitImpl{
		RootPath: gitRoot,
		Author:   author,
		Branch:   branch,
		Remote:   remote,
	}
}

func TraverseToRoot() Git {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	_, err = os.Stat(filepath.Join(cwd, ".git"))
	for os.IsNotExist(err) && filepath.Dir(cwd) != "/" {
		cwd = filepath.Dir(cwd)
		_, err = os.Stat(filepath.Join(cwd, ".git"))
	}
	if filepath.Dir(cwd) == "/" {
		panic("Could not determine Git root for the project")
	}
	return New(cwd)
}

// GitRoot return root path
func (ctx *GitImpl) Root() string {
	return ctx.RootPath
}

// IsWorkTreeClean checks if a worktree is clean (no new changes have been produced)
func (ctx *GitImpl) IsWorkTreeClean() (bool, string, error) {
	_, wt, err := ctx.gitWorkTree()
	if err != nil {
		return false, "", err
	}

	s, err := wt.Status()
	if err != nil {
		return false, "", err
	}

	return s.IsClean(), s.String(), nil
}

// HashShort return first 7 letters of latest commit id
func (ctx *GitImpl) HashShort() (string, error) {
	hash, e := ctx.Hash()
	return hash[:7], e
}

// Hash returns full latest commit id
func (ctx *GitImpl) Hash() (string, error) {
	r, _, err := ctx.gitWorkTree()
	if err != nil {
		return "", err
	}
	plumbingHash, err := r.ResolveRevision(plumbing.Revision("HEAD"))
	if err != nil {
		return "", errors.Wrap(err, "unable to resolve latest git commit hash for repository")
	}
	return plumbingHash.String(), nil
}

// CommitAndPush makes commit and pushes to master
func (ctx *GitImpl) CommitAndPush(msg string) error {
	r, wt, err := ctx.gitWorkTree()
	if err != nil {
		return err
	}

	author := object.Signature{
		Name: ctx.Author,
		When: time.Now(),
	}
	opts := git.CommitOptions{All: true, Author: &author}
	_, err = wt.Commit(msg, &opts)
	if err != nil {
		return err
	}

	pushOpts := git.PushOptions{
		RemoteName: ctx.Remote,
		RefSpecs:   []config.RefSpec{config.RefSpec(ctx.Branch)},
	}
	err = r.Push(&pushOpts)
	if err != nil {
		return err
	}
	return nil
}

func (ctx *GitImpl) gitWorkTree() (*git.Repository, *git.Worktree, error) {
	r, err := git.PlainOpen(ctx.RootPath)
	if err != nil {
		return nil, nil, err
	}
	wt, err := r.Worktree()
	if err != nil {
		return nil, nil, err
	}
	return r, wt, nil
}
