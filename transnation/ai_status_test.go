package transnation

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oneclickvirt/UnlockTests/model"
)

func TestCheckAIRegionalStatusAcceptsDeepSeek202(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cdn-cgi/trace":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("loc=US\n"))
		default:
			w.WriteHeader(http.StatusAccepted)
		}
	}))
	defer server.Close()

	got := checkAIRegionalStatus(server.Client(), aiRegionalProbe{
		name:        "DeepSeek",
		hostname:    "chat.deepseek.com",
		url:         server.URL + "/",
		traceURL:    server.URL + "/cdn-cgi/trace",
		okCodes:     map[int]bool{http.StatusAccepted: true},
		bannedCodes: map[int]bool{http.StatusForbidden: true},
		wafKeywords: defaultAIWAFKeywords(),
	})
	if got.Status != model.StatusYes {
		t.Fatalf("expected 202 Accepted to resolve as Yes, got %#v", got)
	}
	if got.Region != "us" {
		t.Fatalf("expected Cloudflare trace region us, got %q", got.Region)
	}
}

func TestCheckAIRegionalStatusUsesSupportCountryList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cdn-cgi/trace":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("loc=CN\n"))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	got := checkAIRegionalStatus(server.Client(), aiRegionalProbe{
		name:             "Perplexity AI",
		hostname:         "www.perplexity.ai",
		url:              server.URL + "/",
		traceURL:         server.URL + "/cdn-cgi/trace",
		okCodes:          map[int]bool{http.StatusOK: true},
		supportCountries: []string{"us"},
		wafKeywords:      defaultAIWAFKeywords(),
	})
	if got.Status != model.StatusNo {
		t.Fatalf("expected unsupported trace region to resolve as No, got %#v", got)
	}
	if got.Region != "cn" {
		t.Fatalf("expected region cn, got %q", got.Region)
	}
}

func TestCheckAIRegionalStatusUsesRestrictedCountryList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cdn-cgi/trace":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("loc=CN\n"))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	got := checkAIRegionalStatus(server.Client(), aiRegionalProbe{
		name:                "Grok",
		hostname:            "grok.com",
		url:                 server.URL + "/",
		traceURL:            server.URL + "/cdn-cgi/trace",
		okCodes:             map[int]bool{http.StatusOK: true},
		restrictedCountries: []string{"cn"},
		wafKeywords:         defaultAIWAFKeywords(),
	})
	if got.Status != model.StatusNo {
		t.Fatalf("expected restricted trace region to resolve as No, got %#v", got)
	}
	if got.Region != "cn" {
		t.Fatalf("expected region cn, got %q", got.Region)
	}
}

func TestCheckAIRegionalStatusForbiddenCodeUsesTraceRegion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cdn-cgi/trace":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("loc=US\n"))
		default:
			w.WriteHeader(http.StatusForbidden)
		}
	}))
	defer server.Close()

	got := checkAIRegionalStatus(server.Client(), aiRegionalProbe{
		name:                "Perplexity AI",
		hostname:            "www.perplexity.ai",
		url:                 server.URL + "/",
		traceURL:            server.URL + "/cdn-cgi/trace",
		forbiddenCodes:      map[int]bool{http.StatusForbidden: true},
		restrictedCountries: []string{"cn"},
		wafKeywords:         defaultAIWAFKeywords(),
	})
	if got.Status != model.StatusBanned {
		t.Fatalf("expected allowed trace region plus forbidden status to resolve as Banned, got %#v", got)
	}
	if got.Region != "us" {
		t.Fatalf("expected region us, got %q", got.Region)
	}
}

func TestCheckAIRegionalStatusMaps429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cdn-cgi/trace" {
			_, _ = w.Write([]byte("loc=US\n"))
			return
		}
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	got := checkAIRegionalStatus(server.Client(), aiRegionalProbe{
		name:     "Limited AI",
		hostname: "127.0.0.1",
		url:      server.URL + "/",
		traceURL: server.URL + "/cdn-cgi/trace",
	})
	if got.Status != model.StatusRateLimited || got.Region != "us" {
		t.Fatalf("expected structured rate-limit result, got %#v", got)
	}
}

func TestCheckAIRegionalStatusForbiddenDistinguishesRegionFromWAF(t *testing.T) {
	for _, test := range []struct {
		name   string
		region string
		want   string
	}{
		{name: "Grok allowed region", region: "US", want: model.StatusBanned},
		{name: "Grok restricted region", region: "CN", want: model.StatusNo},
		{name: "Perplexity allowed region", region: "US", want: model.StatusBanned},
		{name: "Perplexity restricted region", region: "CN", want: model.StatusNo},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/cdn-cgi/trace" {
					_, _ = w.Write([]byte("loc=" + test.region + "\n"))
					return
				}
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte("Attention Required | Cloudflare"))
			}))
			defer server.Close()

			got := checkAIRegionalStatus(server.Client(), aiRegionalProbe{
				name:                test.name,
				hostname:            "example.test",
				url:                 server.URL + "/",
				traceURL:            server.URL + "/cdn-cgi/trace",
				forbiddenCodes:      map[int]bool{http.StatusForbidden: true},
				restrictedCountries: aiGlobalRestrictedCountries,
				wafKeywords:         defaultAIWAFKeywords(),
			})
			if got.Status != test.want || got.Region != strings.ToLower(test.region) {
				t.Fatalf("got %#v, want status=%q region=%q", got, test.want, strings.ToLower(test.region))
			}
		})
	}
}

func TestCheckAIRegionalStatusSuccessWAFIsBanned(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Just a moment..."))
	}))
	defer server.Close()

	got := checkAIRegionalStatus(server.Client(), aiRegionalProbe{
		name:        "Grok",
		hostname:    "grok.com",
		url:         server.URL + "/",
		traceURL:    server.URL + "/cdn-cgi/trace",
		okCodes:     map[int]bool{http.StatusOK: true},
		wafKeywords: defaultAIWAFKeywords(),
	})
	if got.Status != model.StatusBanned || got.Info != "WAF" {
		t.Fatalf("expected successful WAF page to be banned, got %#v", got)
	}
}

func TestSupportPoeNormalizesCountryCodes(t *testing.T) {
	for _, country := range []string{"US", " cn ", "sg"} {
		if !SupportPoe(country) {
			t.Errorf("SupportPoe(%q) = false, want true", country)
		}
	}
	for _, country := range []string{"", "ZZ", "USA"} {
		if SupportPoe(country) {
			t.Errorf("SupportPoe(%q) = true, want false", country)
		}
	}
}

func TestPoeUsesItsOwnSupportCountryList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cdn-cgi/trace" {
			_, _ = w.Write([]byte("loc=CN\n"))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	got := checkAIRegionalStatus(server.Client(), aiRegionalProbe{
		name:             "Poe",
		hostname:         "poe.com",
		url:              server.URL + "/",
		traceURL:         server.URL + "/cdn-cgi/trace",
		okCodes:          map[int]bool{http.StatusOK: true},
		supportCountries: poeSupportCountries,
		wafKeywords:      defaultAIWAFKeywords(),
	})
	if got.Status != model.StatusYes || got.Region != "cn" {
		t.Fatalf("Poe CN result = %#v, want Yes with cn region", got)
	}
}
