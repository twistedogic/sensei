package model

import "context"

type ctxKey string

const metaKey ctxKey = "meta"

type Metadata struct {
	Temperature      float32
	OutputTokenLimit int
}

func MetaFromContext(ctx context.Context) Metadata {
	val := ctx.Value(metaKey)
	meta, ok := val.(Metadata)
	if ok {
		return meta
	}
	return Metadata{}
}

func WithMeta(ctx context.Context, meta Metadata) context.Context {
	return context.WithValue(ctx, metaKey, meta)
}
