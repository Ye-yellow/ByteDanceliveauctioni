package searchindex

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEmbeddingClientEmbedsDashScopeCompatibleResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/compatible-mode/v1/embeddings" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		var req struct {
			Model      string   `json:"model"`
			Input      []string `json:"input"`
			Dimensions int      `json:"dimensions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "text-embedding-v4" || req.Dimensions != 3 || len(req.Input) != 2 {
			t.Fatalf("unexpected request: %+v", req)
		}
		_, _ = w.Write([]byte(`{
			"data": [
				{"index": 1, "embedding": [0.4, 0.5, 0.6]},
				{"index": 0, "embedding": [0.1, 0.2, 0.3]}
			]
		}`))
	}))
	defer server.Close()

	client := NewEmbeddingClient(EmbeddingConfig{
		Provider:   "dashscope",
		BaseURL:    server.URL + "/compatible-mode/v1",
		Model:      "text-embedding-v4",
		APIKey:     "test-key",
		Dimensions: 3,
		Timeout:    time.Second,
	})

	embeddings, err := client.Embed(context.Background(), []string{"翡翠手镯", "送礼收藏"})
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(embeddings) != 2 || embeddings[0][0] != 0.1 || embeddings[1][0] != 0.4 {
		t.Fatalf("unexpected embeddings: %#v", embeddings)
	}
}

func TestEmbeddingClientRequiresConfiguration(t *testing.T) {
	client := NewEmbeddingClient(EmbeddingConfig{Provider: "dashscope"})
	if client.Configured() {
		t.Fatal("client should not be configured without api key")
	}
}
