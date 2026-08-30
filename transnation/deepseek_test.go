package transnation

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oneclickvirt/UnlockTests/model"
	"github.com/oneclickvirt/UnlockTests/utils"
)

func TestCheckDeepSeekUsesHostAndParsesRegion(t *testing.T) {
	var gotHost string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost = r.Host
		_, _ = w.Write([]byte(`<meta name="region" content="US">`))
	}))
	defer server.Close()

	got := checkDeepSeek(server.Client(), server.URL+"/sign_in", deepSeekHost)
	if got.Status != model.StatusYes || got.Region != "us" {
		t.Fatalf("unexpected result: %#v", got)
	}
	if gotHost != deepSeekHost {
		t.Fatalf("Host = %q, want %q", gotHost, deepSeekHost)
	}
}

func TestCheckDeepSeekStatusMapping(t *testing.T) {
	tests := map[string]struct {
		code int
		body string
		want string
	}{
		"forbidden":     {code: http.StatusForbidden, want: model.StatusNo},
		"waf forbidden": {code: http.StatusForbidden, body: "Attention Required | Cloudflare", want: model.StatusBanned},
		"rate limited":  {code: http.StatusTooManyRequests, want: model.StatusRateLimited},
		"legal":         {code: http.StatusUnavailableForLegalReasons, want: model.StatusNo},
		"accepted":      {code: http.StatusAccepted, want: model.StatusYes},
		"redirect":      {code: http.StatusTemporaryRedirect, want: model.StatusYes},
		"unexpected":    {code: http.StatusBadGateway, want: model.StatusUnexpected},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.code)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			got := checkDeepSeek(server.Client(), server.URL, "chat.deepseek.com")
			if got.Status != tt.want {
				t.Fatalf("got %#v, want status %q", got, tt.want)
			}
		})
	}
}

func TestCheckDeepSeekSetsTLSSNI(t *testing.T) {
	var gotSNI string
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil {
			gotSNI = r.TLS.ServerName
		}
		_, _ = w.Write([]byte(`<meta name="region" content="JP">`))
	}))
	server.StartTLS()
	defer server.Close()

	client := server.Client()
	transport := client.Transport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // test certificate is intentionally self-signed
	client.Transport = transport
	got := checkDeepSeek(client, server.URL, deepSeekHost)
	if got.Status != model.StatusYes {
		t.Fatalf("unexpected TLS result: %#v", got)
	}
	if !strings.EqualFold(gotSNI, deepSeekHost) {
		t.Fatalf("TLS SNI = %q, want %q", gotSNI, deepSeekHost)
	}
}

func TestCheckDeepSeekSkipsIPv4LiteralForIPv6Client(t *testing.T) {
	oldVersion := utils.GetDNSIPVersion()
	utils.SetDNSIPVersion("ipv6")
	defer utils.SetDNSIPVersion(oldVersion)

	got := checkDeepSeek(&http.Client{Transport: &http.Transport{}}, "https://116.205.40.114/sign_in", deepSeekHost)
	if got.Status != model.StatusNoIPv6 {
		t.Fatalf("got %#v, want %s", got, model.StatusNoIPv6)
	}
}
