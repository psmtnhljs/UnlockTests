package jp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oneclickvirt/UnlockTests/model"
)

func TestCheckDMMUsesGraphQLForeignAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/graphql" {
			t.Fatalf("request = %s %s, want POST /graphql", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if !strings.Contains(string(body), "isForeignAccess") {
			t.Fatalf("request body = %q, want isForeignAccess query", body)
		}
		_, _ = w.Write([]byte(`{"data":{"client":{"isForeignAccess":true}}}`))
	}))
	defer server.Close()

	got := checkDMM(server.Client(), server.URL+"/graphql")
	if got.Name != "DMM" || got.Status != model.StatusNo {
		t.Fatalf("unexpected foreign result: %#v", got)
	}
}

func TestCheckDMMMapsAllowedAndMalformedResponses(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
		want string
	}{
		{name: "allowed", body: `{"data":{"client":{"isForeignAccess":false}}}`, code: http.StatusOK, want: model.StatusYes},
		{name: "missing field", body: `{"data":{"client":{}}}`, code: http.StatusOK, want: model.StatusUnexpected},
		{name: "rate limited", body: `{}`, code: http.StatusTooManyRequests, want: model.StatusRateLimited},
		{name: "forbidden", body: `{}`, code: http.StatusForbidden, want: model.StatusNo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.code)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			got := checkDMM(server.Client(), server.URL)
			if got.Status != tt.want {
				t.Fatalf("got %#v, want status %q", got, tt.want)
			}
		})
	}
}
