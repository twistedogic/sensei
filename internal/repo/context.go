package repo

import "context"

type ctxKey string

const (
	commitKey ctxKey = "commit"
	diffKey   ctxKey = "diff"

	headRef = "HEAD"
)

func (c Commit) IsHead() bool { return c == headRef }

type DiffOption struct {
	From, To Commit
}

func (d DiffOption) IsZero() bool { return d.From == "" && d.To == "" }

func WithCommit(ctx context.Context, commit Commit) context.Context {
	return context.WithValue(ctx, commitKey, commit)
}

func CommitFromContext(ctx context.Context) Commit {
	val := ctx.Value(commitKey)
	commit, ok := val.(Commit)
	if ok {
		return commit
	}
	return headRef
}

func WithDiff(ctx context.Context, from, to Commit) context.Context {
	return context.WithValue(ctx, diffKey, DiffOption{From: from, To: to})
}

func DiffFromContext(ctx context.Context) DiffOption {
	val := ctx.Value(diffKey)
	diff, ok := val.(DiffOption)
	if ok {
		return diff
	}
	return DiffOption{}
}
