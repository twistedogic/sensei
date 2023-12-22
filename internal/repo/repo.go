package repo

import (
	"context"
	"io"
)

type Commit string

func (c Commit) String() string { return string(c) }

type File interface {
	Name() string
	io.Reader
	io.Closer
}

type Differ interface {
	Diff(context.Context, string) (string, string, error)
	DiffPatch(context.Context, io.Writer) error
}

type Committer interface {
	Add(context.Context, []File) error
	Commit(context.Context, string) (Commit, error)
}

type Reader interface {
	Read(context.Context, string, io.Writer) error
	Head(context.Context, string, io.Writer) error
	List(context.Context) ([]string, error)
}

type Repo interface {
	Reader
	Committer
	Differ
}
