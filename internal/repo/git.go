package repo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/diff"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const author = "sensei"

func gitSignature(now time.Time) *object.Signature {
	return &object.Signature{
		Name: author,
		When: now,
	}
}

func gitCommitHash(commit Commit) plumbing.Hash {
	return plumbing.NewHash(commit.String())
}

type gitRepo struct {
	repo *git.Repository
}

func walkFs(fs billy.Filesystem, root string, f func(string)) error {
	queue := []string{root}
	var current string
	for len(queue) != 0 {
		current, queue = queue[0], queue[1:]
		infos, err := fs.ReadDir(current)
		if err != nil {
			return err
		}
		for _, info := range infos {
			path := filepath.Join(current, info.Name())
			if info.IsDir() {
				queue = append(queue, path)
			} else {
				f(path)
			}
		}
	}
	return nil
}

type readFunc func(context.Context, string, io.Writer) error

func readContent(ctx context.Context, path string, f readFunc) (string, error) {
	buf := &bytes.Buffer{}
	err := f(ctx, path, buf)
	return buf.String(), err
}

func FromPath(path string) (Repo, error) {
	repo, err := git.PlainOpen(path)
	return gitRepo{repo: repo}, err
}

func (g gitRepo) fs() (billy.Filesystem, error) {
	worktree, err := g.repo.Worktree()
	return worktree.Filesystem, err
}

func (g gitRepo) commitObject(commit Commit) (*object.Commit, error) {
	o, err := g.repo.CommitObject(gitCommitHash(commit))
	if err != nil {
		return nil, fmt.Errorf("get commit object %s: %w", commit, err)
	}
	return o, nil
}

func (g gitRepo) readFromWorktree(path string, w io.Writer) error {
	fs, err := g.fs()
	if err != nil {
		return err
	}
	f, err := fs.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(w, f); err != nil {
		return err
	}
	return nil
}

func (g gitRepo) readFromCommit(commit Commit, path string, w io.Writer) error {
	o, err := g.commitObject(commit)
	if err != nil {
		return err
	}
	blob, err := o.File(path)
	if err != nil {
		return err
	}
	f, err := blob.Reader()
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(w, f); err != nil {
		return err
	}
	return nil
}

func (g gitRepo) Read(ctx context.Context, path string, w io.Writer) error {
	commit := CommitFromContext(ctx)
	if commit.IsHead() {
		return g.readFromWorktree(path, w)
	}
	return g.readFromCommit(commit, path, w)
}

func (g gitRepo) status() (git.Status, error) {
	worktree, err := g.repo.Worktree()
	if err != nil {
		return nil, err
	}
	return worktree.Status()
}

func (g gitRepo) Head(ctx context.Context, path string, w io.Writer) error {
	head, err := g.repo.Head()
	if err != nil {
		return err
	}
	commit := Commit(head.Hash().String())
	readCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	return g.Read(WithCommit(readCtx, commit), path, w)
}

func (g gitRepo) diffFromWorktree(ctx context.Context, path string) (from, to string, err error) {
	from, err = readContent(ctx, path, g.Head)
	if err != nil {
		return
	}
	to, err = readContent(ctx, path, g.Read)
	return
}

func (g gitRepo) diffFromCommits(opt DiffOption, path string) (from, to string, err error) {
	from, err = readContent(WithCommit(context.Background(), opt.From), path, g.Read)
	if err != nil {
		return
	}
	to, err = readContent(WithCommit(context.Background(), opt.To), path, g.Read)
	return
}

func (g gitRepo) Diff(ctx context.Context, path string) (from, to string, err error) {
	if opt := DiffFromContext(ctx); !opt.IsZero() {
		return g.diffFromCommits(opt, path)
	}
	return g.diffFromWorktree(ctx, path)
}

func (g gitRepo) diffPatchFromWorktree(ctx context.Context, w io.Writer) error {
	patcher := diffmatchpatch.New()
	status, err := g.status()
	if err != nil || status.IsClean() {
		return err
	}
	diffs := make([]string, 0, len(status)*2)
	for path, fileStatus := range status {
		switch fileStatus.Staging {
		case git.Untracked, git.Added:
			to, err := readContent(ctx, path, g.Read)
			if err != nil {
				return err
			}
			patches := patcher.PatchMake(diff.Do("", to))
			diffs = append(diffs, path, patcher.PatchToText(patches))
		case git.Modified:
			from, to, err := g.Diff(ctx, path)
			if err != nil {
				return err
			}
			patches := patcher.PatchMake(diff.Do(from, to))
			diffs = append(diffs, path, patcher.PatchToText(patches))
		}
	}
	if _, err := w.Write([]byte(strings.Join(diffs, "\n"))); err != nil {
		return err
	}
	return nil
}

func (g gitRepo) diffPatchCommits(opt DiffOption, w io.Writer) error {
	fromCommit, err := g.commitObject(opt.From)
	if err != nil {
		return err
	}
	toCommit, err := g.commitObject(opt.To)
	if err != nil {
		return err
	}
	patch, err := fromCommit.Patch(toCommit)
	if err != nil {
		return err
	}
	return patch.Encode(w)
}

func (g gitRepo) DiffPatch(ctx context.Context, w io.Writer) error {
	diffOpt := DiffFromContext(ctx)
	if diffOpt.IsZero() {
		return g.diffPatchFromWorktree(ctx, w)
	}
	return g.diffPatchCommits(diffOpt, w)
}

func (g gitRepo) listFromWorktree() ([]string, error) {
	fs, err := g.fs()
	if err != nil {
		return nil, err
	}
	paths := []string{}
	if err := walkFs(fs, ".", func(name string) {
		paths = append(paths, name)
	}); err != nil {
		return nil, err
	}
	return paths, nil
}

func (g gitRepo) listFromCommit(commit Commit) ([]string, error) {
	o, err := g.commitObject(commit)
	if err != nil {
		return nil, err
	}
	iter, err := o.Files()
	if err != nil {
		return nil, err
	}
	files := []string{}
	if err := iter.ForEach(func(f *object.File) error {
		files = append(files, f.Name)
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}

func (g gitRepo) List(ctx context.Context) ([]string, error) {
	if commit := CommitFromContext(ctx); !commit.IsHead() {
		return g.listFromCommit(commit)
	}
	return g.listFromWorktree()
}

func (g gitRepo) add(tree *git.Worktree, path string, content []byte) error {
	fs := tree.Filesystem
	if len(content) == 0 {
		return fs.Remove(path)
	}
	file, err := fs.Create(path)
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if _, err := tree.Add(path); err != nil {
		return err
	}
	return nil
}

func (g gitRepo) stage(tree *git.Worktree, files []File) error {
	for _, f := range files {
		path := f.Name()
		b, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		if err := g.add(tree, path, b); err != nil {
			return err
		}
	}
	return nil
}

func (g gitRepo) Add(ctx context.Context, files []File) error {
	tree, err := g.repo.Worktree()
	if err != nil {
		return err
	}
	return g.stage(tree, files)
}

func (g gitRepo) Commit(ctx context.Context, message string) (Commit, error) {
	tree, err := g.repo.Worktree()
	if err != nil {
		return "", err
	}
	hash, err := tree.Commit(message, &git.CommitOptions{
		All: true, Author: gitSignature(time.Now()),
	})
	if err != nil {
		return "", err
	}
	return Commit(hash.String()), nil
}
