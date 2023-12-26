package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/jmorganca/ollama/api"
)

const (
	defaultLocalAddr = "http://127.0.0.1:11434"

	infoPath      = "/api/show"
	embeddingPath = "/api/embeddings"
	generatePath  = "/api/generate"
)

type ollamaModel struct {
	client        *http.Client
	addr, modelId string
}

func NewOllamaModel(addr, modelId string) (Model, error) {
	if _, err := url.Parse(addr); err != nil {
		return nil, err
	}
	m := ollamaModel{addr: addr, modelId: modelId, client: http.DefaultClient}
	_, err := m.info()
	return m, err
}

func NewLocalOllamaModel(modelId string) (Model, error) {
	return NewOllamaModel(defaultLocalAddr, modelId)
}

func (o ollamaModel) postJSON(ctx context.Context, target string, postBody, body interface{}) error {
	b, err := json.Marshal(postBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", target, bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	res, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if status := res.StatusCode; status != http.StatusOK {
		return fmt.Errorf("POST to %s failed with status code %d", target, status)
	}
	return json.NewDecoder(res.Body).Decode(body)
}

func (o ollamaModel) info() (*api.ShowResponse, error) {
	req := api.ShowRequest{Name: o.modelId}
	res := &api.ShowResponse{}
	target := o.addr + infoPath
	err := o.postJSON(context.Background(), target, req, res)
	return res, err
}

func (o ollamaModel) Embeddings(ctx context.Context, prompt string) ([]float64, error) {
	req := api.EmbeddingRequest{
		Model:  o.modelId,
		Prompt: prompt,
	}
	res := &api.EmbeddingResponse{}
	target := o.addr + embeddingPath
	err := o.postJSON(ctx, target, req, res)
	return res.Embedding, err
}

func (o ollamaModel) Prompt(ctx context.Context, m Message, w io.Writer) error {
	stream := false
	req := api.GenerateRequest{Model: o.modelId, Stream: &stream}
	if m.System != "" {
		req.System = m.System
	}
	if m.Context != "" {
		req.Prompt = m.Context + " " + m.User
	} else {
		req.Prompt = m.User
	}
	res := &api.GenerateResponse{}
	target := o.addr + generatePath
	if err := o.postJSON(ctx, target, req, res); err != nil {
		return err
	}
	_, err := w.Write([]byte(res.Response))
	return err
}
