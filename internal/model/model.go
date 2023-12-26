package model

import (
	"context"
	"io"
)

type Message struct {
	System, User, Context, Template string
	Metadata                        map[string]string
}

type Prompter interface {
	Prompt(context.Context, Message, io.Writer) error
}

type Embedder interface {
	Embeddings(context.Context, string) ([]float64, error)
}

type Model interface {
	Embedder
	Prompter
}
