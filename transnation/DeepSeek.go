package transnation

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	req "github.com/imroc/req/v3"
	"github.com/oneclickvirt/UnlockTests/model"
	"github.com/oneclickvirt/UnlockTests/utils"
)

const (
	deepSeekHost        = "chat.deepseek.com"
	deepSeekDefaultIP   = "116.205.40.114"
	deepSeekPath        = "/sign_in"
	deepSeekIPEnv       = "UNLOCKTESTS_DEEPSEEK_IP"
	deepSeekMaxBodySize = 2 << 20
)

var deepseekRegionRegex = regexp.MustCompile(`(?is)<meta\b[^>]*\bname\s*=\s*["']region["'][^>]*\bcontent\s*=\s*["']([^"']+)["']`)

var deepseekRegionReverseRegex = regexp.MustCompile(`(?is)<meta\b[^>]*\bcontent\s*=\s*["']([^"']+)["'][^>]*\bname\s*=\s*["']region["']`)

var deepSeekWAFMarkers = []string{
	"attention required",
	"cf-chl-",
	"checking your browser",
	"just a moment",
	"access denied",
	"request blocked",
	"cloudflare ray id",
}

func DeepSeek(c *http.Client) model.Result {
	endpoint := strings.TrimSpace(os.Getenv(deepSeekIPEnv))
	return checkDeepSeek(c, endpoint, deepSeekHost)
}

// checkDeepSeek performs the DeepSeek probe using req through utils.ReqDefault.
// endpoint may be a complete URL (useful for tests) or an IP/host; when empty,
// the configured direct CN IP is used. The optional host argument overrides
// the Host header and TLS SNI for an injected endpoint.
func checkDeepSeek(c *http.Client, endpoint string, hostOverride ...string) model.Result {
	const name = "DeepSeek"
	if c == nil {
		return model.Result{Name: name}
	}

	host := deepSeekHost
	if len(hostOverride) > 0 && strings.TrimSpace(hostOverride[0]) != "" {
		host = strings.TrimSpace(hostOverride[0])
	}
	endpoint = deepSeekEndpoint(endpoint)
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return model.Result{Name: name, Status: model.StatusErr, Err: fmt.Errorf("invalid DeepSeek endpoint")}
	}
	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = deepSeekPath
	}
	endpoint = parsed.String()
	if utils.IsIPv6Client(c) {
		if ip := net.ParseIP(parsed.Hostname()); ip != nil && ip.To4() != nil {
			return model.Result{Name: name, Status: model.StatusNoIPv6, Info: "IPv4-only endpoint"}
		}
	}

	// ReqDefault retains the caller's http.Transport (including proxy and
	// IPv4/IPv6 dial settings) while avoiding a fingerprint-specific handshake
	// that would otherwise use the IP as SNI. Clone the TLS config so this
	// per-request SNI override cannot mutate the shared transport.
	client := utils.ReqDefault(c)
	client.SetRedirectPolicy(req.NoRedirectPolicy())
	request := client.R().
		SetHeader("Host", host).
		SetHeader("User-Agent", model.UA_Browser).
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	if parsed.Scheme == "https" {
		config := client.GetTLSClientConfig()
		if config == nil {
			config = &tls.Config{}
		} else {
			config = config.Clone()
		}
		config.ServerName = host
		client.SetTLSClientConfig(config)
	}
	resp, err := request.Get(endpoint)
	if err != nil {
		return utils.HandleNetworkError(c, host, err, name)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, deepSeekMaxBodySize+1))
	if readErr != nil {
		return utils.HandleNetworkError(c, host, readErr, name)
	}
	if len(body) > deepSeekMaxBodySize {
		return model.Result{Name: name, Status: model.StatusErr, Err: fmt.Errorf("DeepSeek response exceeds %d bytes", deepSeekMaxBodySize)}
	}

	switch {
	case resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest:
		region := parseDeepSeekRegion(body)
		return model.Result{Name: name, Status: model.StatusYes, Region: region}
	case resp.StatusCode == http.StatusForbidden:
		if deepSeekIsWAFBody(body) {
			return model.Result{Name: name, Status: model.StatusBanned, Info: "WAF"}
		}
		return model.Result{Name: name, Status: model.StatusNo}
	case resp.StatusCode == http.StatusTooManyRequests:
		return model.Result{Name: name, Status: model.StatusRateLimited, Info: "HTTP 429"}
	case resp.StatusCode == http.StatusUnavailableForLegalReasons:
		return model.Result{Name: name, Status: model.StatusNo}
	default:
		return model.Result{Name: name, Status: model.StatusUnexpected,
			Err: fmt.Errorf("DeepSeek returned HTTP %d", resp.StatusCode)}
	}
}

func deepSeekEndpoint(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = deepSeekDefaultIP
	}
	if strings.Contains(value, "://") {
		return value
	}
	if ip := net.ParseIP(strings.Trim(value, "[]")); ip != nil && ip.To4() == nil {
		value = "[" + strings.Trim(value, "[]") + "]"
	}
	return "https://" + value + deepSeekPath
}

func parseDeepSeekRegion(body []byte) string {
	match := deepseekRegionRegex.FindSubmatch(body)
	if len(match) < 2 {
		match = deepseekRegionReverseRegex.FindSubmatch(body)
	}
	if len(match) < 2 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(string(match[1])))
}

func deepSeekIsWAFBody(body []byte) bool {
	text := strings.ToLower(string(body))
	for _, marker := range deepSeekWAFMarkers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
