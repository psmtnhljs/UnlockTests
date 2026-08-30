package transnation

import (
	"net/http"
	"strings"

	"github.com/oneclickvirt/UnlockTests/model"
)

// Poe publishes a broader availability list than the other AI providers.
// Keep the list local so Poe is not incorrectly treated as unavailable in a
// country that is restricted by another provider.
const poeSupportCountryCodes = `AF AL DZ AS AD AO AI AG AR AM AW AU AT AZ BS BH BD BB BY BE BZ BJ BM BT BO BA BW BR VG BN BG BF BI KH CM CA CV KY CF TD IO CL CN CX CC CO KM CG CD CK CR CI HR CU CY CZ DK DJ DM DO EC EG SV GQ ER EE SZ ET FK FO FJ FI FR GF PF GA GM GE DE GH GI GR GL GD GP GU GT GG GN GW GY HT HN HK HU IS IN ID IR IQ IE IM IL IT JM JP JE JO KZ KE KI KW KG LA LV LB LS LR LY LI LT LU MO MG MW MY MV ML MT MH MQ MR MU YT MX FM MD MC MN ME MS MA MZ MM NA NR NP NL NC NZ NI NE NG NU NF KP MK MP NO OM PK PW PS PA PG PY PE PH PN PL PT PR QA RE RO RU RW WS SM ST SA SN RS SC SL SG SK SI GS SB SO ZA KR ES LK BL SH KN LC MF PM VC SD SR SJ SE CH SY TW TJ TZ TH TL TG TK TO TT TN TR TM TC TV VI UG UA AE GB US UY UZ VU VA VE VN WF YE ZM ZW`

var poeSupportCountries = func() []string {
	codes := strings.Fields(poeSupportCountryCodes)
	result := make([]string, 0, len(codes))
	for _, code := range codes {
		result = append(result, strings.ToLower(code))
	}
	return result
}()

var poeSupportCountrySet = func() map[string]struct{} {
	result := make(map[string]struct{}, len(poeSupportCountries))
	for _, code := range poeSupportCountries {
		result[code] = struct{}{}
	}
	return result
}()

// SupportPoe reports whether a two-letter country code is included in Poe's
// published supported-country list. Comparison tolerates case and whitespace
// differences in trace responses.
func SupportPoe(loc string) bool {
	_, ok := poeSupportCountrySet[strings.ToLower(strings.TrimSpace(loc))]
	return ok
}

func Poe(c *http.Client) model.Result {
	return checkAIRegionalStatus(c, aiRegionalProbe{
		name:             "Poe",
		hostname:         "poe.com",
		url:              "https://poe.com/",
		traceURL:         "https://poe.com/cdn-cgi/trace",
		noRedirect:       true,
		okCodes:          map[int]bool{http.StatusOK: true, http.StatusMovedPermanently: true, http.StatusFound: true, http.StatusTemporaryRedirect: true, http.StatusPermanentRedirect: true},
		noCodes:          map[int]bool{http.StatusForbidden: true, http.StatusUnavailableForLegalReasons: true},
		supportCountries: poeSupportCountries,
		wafKeywords:      defaultAIWAFKeywords(),
	})
}
