package geo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestHTTPClient(t *testing.T, server *httptest.Server) *http.Client {
	t.Helper()

	parsedURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse server url: %v", err)
	}

	proxyClient := server.Client()
	baseTransport := proxyClient.Transport
	t.Cleanup(func() {
		if transport, ok := baseTransport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	})

	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			clone := req.Clone(req.Context())
			clone.URL.Scheme = parsedURL.Scheme
			clone.URL.Host = parsedURL.Host
			clone.Host = parsedURL.Host
			clone.RequestURI = ""
			return proxyClient.Do(clone)
		}),
	}
}

func TestDGISClientRouteMatrix(t *testing.T) {
	apiKey := "test-api-key"

	tests := []struct {
		name        string
		handler     func(t *testing.T, w http.ResponseWriter, r *http.Request)
		wantDist    int
		wantDur     int
		wantErr     bool
		errContains string
	}{
		{
			name: "ok",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("failed to read body: %v", err)
				}
				payload := map[string]any{}
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatalf("failed to unmarshal body: %v", err)
				}
				if points, ok := payload["points"].([]any); !ok || len(points) != 2 {
					t.Fatalf("unexpected points payload: %v", payload["points"])
				}
				if sources, ok := payload["sources"].([]any); !ok || len(sources) != 1 || sources[0].(float64) != 0 {
					t.Fatalf("unexpected sources payload: %v", payload["sources"])
				}
				if targets, ok := payload["targets"].([]any); !ok || len(targets) != 1 || targets[0].(float64) != 1 {
					t.Fatalf("unexpected targets payload: %v", payload["targets"])
				}
				if transport, ok := payload["transport"].(string); !ok || transport != "driving" {
					t.Fatalf("unexpected transport: %v", payload["transport"])
				}
				if typ, ok := payload["type"].(string); !ok || typ != "jam" {
					t.Fatalf("unexpected type: %v", payload["type"])
				}

				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{"routes":[{"status":"OK","distance":1234,"duration":567}]}`)
			},
			wantDist: 1234,
			wantDur:  567,
		},
		{
			name: "no content",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr:     true,
			errContains: "route not found",
		},
		{
			name: "status error",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				w.WriteHeader(http.StatusBadRequest)
				io.WriteString(w, "bad request")
			},
			wantErr:     true,
			errContains: "400",
		},
		{
			name: "empty routes",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				io.Copy(io.Discard, r.Body)
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{"routes":[]}`)
			},
			wantErr:     true,
			errContains: "empty routes",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("unexpected method: %s", r.Method)
				}
				if r.URL.Path != "/get_dist_matrix" {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				if got := r.URL.Query().Get("key"); got != apiKey {
					t.Fatalf("unexpected key param: %s", got)
				}
				if got := r.URL.Query().Get("version"); got != "2.0" {
					t.Fatalf("unexpected version param: %s", got)
				}
				if got := r.URL.Query().Get("response_format"); got != "json" {
					t.Fatalf("unexpected response_format param: %s", got)
				}
				if got := r.Header.Get("Content-Type"); got != "application/json" {
					t.Fatalf("unexpected content-type: %s", got)
				}
				tt.handler(t, w, r)
			}))
			defer server.Close()

			httpClient := newTestHTTPClient(t, server)
			client := NewDGISClient(httpClient, apiKey, "")

			dist, dur, err := client.RouteMatrix(context.Background(), 10.0, 20.0, 30.0, 40.0)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dist != tt.wantDist || dur != tt.wantDur {
				t.Fatalf("unexpected result: dist=%d dur=%d", dist, dur)
			}
		})
	}
}
