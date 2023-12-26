package model

import (
	"testing"
)

func Test_ollamaModel(t *testing.T) {
	model, err := NewLocalOllamaModel("mistral")
	if err != nil {
		t.Skip()
	}
	RunPrompterTest(t, model)
	RunEmbedderTest(t, model)
}
