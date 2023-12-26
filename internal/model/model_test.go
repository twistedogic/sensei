package model

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func RunPrompterTest(t *testing.T, prompter Prompter) {
	t.Run("prompt", func(t *testing.T) {
		cases := map[string]struct {
			message Message
			want    string
		}{
			"base": {
				message: Message{
					User: "What is the answer for 12 + 30? Answer with number only.",
				},
				want: "42",
			},
			"system": {
				message: Message{
					System: "You are a 5 years-old boy.",
					User:   "Write your age in integer:",
				},
				want: "5",
			},
			"context": {
				message: Message{
					Context: "Luke Skywalker is Anakin Skywalker's son.",
					User:    "Write the name of Luke Skywalker's father:",
				},
				want: "Anakin",
			},
		}
		for name := range cases {
			tc := cases[name]
			t.Run(name, func(t *testing.T) {
				buf := &bytes.Buffer{}
				require.NoError(t, prompter.Prompt(context.TODO(), tc.message, buf), "prompt")
				require.Contains(t, buf.String(), tc.want, "prompt response")
			})
		}
	})
}

func RunEmbedderTest(t *testing.T, e Embedder) {
	t.Run("embeddings", func(t *testing.T) {
		cases := []string{
			"This is a test for a embeddings",
		}
		for _, prompt := range cases {
			once, err := e.Embeddings(context.TODO(), prompt)
			require.NoError(t, err, "embeddings")
			require.NotEqual(t, 0, len(once))
			twice, err := e.Embeddings(context.TODO(), prompt)
			require.NoError(t, err, "embeddings")
			require.NotEqual(t, 0, len(twice))
			require.Equal(t, once, twice, "repeat embeddings")
		}
	})
}
