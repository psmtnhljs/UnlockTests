package transnation

import (
	"net/http"
	"strings"

	"github.com/oneclickvirt/UnlockTests/model"
	"github.com/oneclickvirt/UnlockTests/utils"
)

// SupportGrok reports whether the trace country is not one of Grok's
// explicitly restricted regions. It is exported for callers that need to
// explain a 403 result without issuing another request.
func SupportGrok(loc string) bool {
	loc = strings.ToLower(strings.TrimSpace(loc))
	return loc != "" && !utils.GetRegion(loc, aiGlobalRestrictedCountries)
}

func Grok(c *http.Client) model.Result {
	return checkAIRegionalStatus(c, aiRegionalProbe{
		name:                "Grok",
		hostname:            "grok.com",
		url:                 "https://grok.com/",
		traceURL:            "https://grok.com/cdn-cgi/trace",
		okCodes:             map[int]bool{http.StatusOK: true, http.StatusAccepted: true, http.StatusFound: true, http.StatusTemporaryRedirect: true, http.StatusPermanentRedirect: true},
		forbiddenCodes:      map[int]bool{http.StatusForbidden: true},
		restrictedCountries: aiGlobalRestrictedCountries,
		wafKeywords:         defaultAIWAFKeywords(),
	})
}
