package repo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

type mockFile struct {
	name string
	*bytes.Buffer
}

func (m mockFile) Name() string { return m.name }
func (m mockFile) Close() error { return nil }

func newFile(name, content string) File {
	return mockFile{name: name, Buffer: bytes.NewBuffer([]byte(content))}
}

type step struct {
	files                    []File
	paths                    []string
	read, wantRead, wantDiff string
}

func checkReadContent(t *testing.T, f func(io.Writer) error, format string, arg ...interface{}) string {
	buf := &bytes.Buffer{}
	require.NoErrorf(t, f(buf), format, arg...)
	return buf.String()
}

func checkMatchContent(t *testing.T, want string, f func(io.Writer) error, format string, arg ...interface{}) {
	got := checkReadContent(t, f, format, arg...)
	require.Emptyf(t, cmp.Diff(want, got), format, arg...)
}

func checkContainContent(t *testing.T, want string, f func(io.Writer) error, format string, arg ...interface{}) {
	got := checkReadContent(t, f, format, arg...)
	require.Containsf(t, got, want, format, arg...)
}

func checkState(t *testing.T, ctx context.Context, repo Repo, change step) {
	commit := CommitFromContext(ctx)
	paths, err := repo.List(ctx)
	require.NoErrorf(t, err, "list commit %v", commit)
	sort.Strings(paths)
	sort.Strings(change.paths)
	require.Emptyf(
		t, cmp.Diff(change.paths, paths),
		"list paths with commit %v", commit,
	)
	checkMatchContent(t, change.wantRead, func(w io.Writer) error {
		return repo.Read(ctx, change.read, w)
	}, "read %s from %s", change.read, commit)
}

func repoTest(t *testing.T, repo Repo) {
	changes := []step{
		{
			files: []File{
				newFile("test.txt", "something"),
				newFile("dir/test.txt", "something"),
			},
			paths: []string{"test.txt", "dir/test.txt"},
			read:  "dir/test.txt", wantRead: "something",
			wantDiff: `+something`,
		},
		{
			files: []File{
				newFile("test.txt", ""),
				newFile("dir/test.txt", "other thing"),
			},
			paths: []string{"dir/test.txt"},
			read:  "dir/test.txt", wantRead: "other thing",
			wantDiff: `-something`,
		},
	}
	commits := make([]Commit, len(changes))
	for i, change := range changes {
		ctx := context.TODO()
		require.NoError(t, repo.Add(ctx, change.files), "add")
		checkContainContent(t, change.wantDiff, func(w io.Writer) error {
			return repo.DiffPatch(ctx, w)
		}, "diff patch")
		msg := fmt.Sprintf("change %d", i)
		commit, err := repo.Commit(ctx, msg)
		require.NoError(t, err, "commit")
		commits[i] = commit
		checkState(t, ctx, repo, change)
	}
	for i := range changes {
		if i != 0 {
			from, to := commits[i-1], commits[i]
			ctx := WithDiff(context.TODO(), from, to)
			checkContainContent(t, changes[i].wantDiff, func(w io.Writer) error {
				return repo.DiffPatch(ctx, w)
			}, "diff patch")
		}
		idx := len(changes) - 1 - i
		change, commit := changes[idx], commits[idx]
		ctx := WithCommit(context.TODO(), commit)
		checkState(t, ctx, repo, change)
	}
}
