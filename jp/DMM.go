package jp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/oneclickvirt/UnlockTests/model"
	"github.com/oneclickvirt/UnlockTests/utils"
)

const (
	dmmGraphQLURL  = "https://api.tv.dmm.com/graphql"
	dmmGraphQLBody = `{"query":"query FetchClient { client { isForeignAccess } }"}`
)

type dmmGraphQLResponse struct {
	Data struct {
		Client struct {
			IsForeignAccess *bool `json:"isForeignAccess"`
		} `json:"client"`
	} `json:"data"`
	Errors []json.RawMessage `json:"errors"`
}

// DMM uses the TV GraphQL endpoint to determine whether the caller is being
// treated as a foreign client. The previous bitcoin.dmm.com page is unrelated
// to the streaming service and can return a misleading successful page.
func DMM(c *http.Client) model.Result {
	return checkDMM(c, dmmGraphQLURL)
}

func checkDMM(c *http.Client, endpoint string) model.Result {
	name := "DMM"
	hostname := "api.tv.dmm.com"
	if c == nil {
		return model.Result{Name: name}
	}
	client := utils.Req(c)
	resp, err := client.R().
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetBodyString(dmmGraphQLBody).
		Post(endpoint)
	if err != nil {
		return utils.HandleNetworkError(c, hostname, err, name)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return model.Result{Name: name, Status: model.StatusRateLimited, Info: "HTTP 429"}
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnavailableForLegalReasons {
		return model.Result{Name: name, Status: model.StatusNo}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return model.Result{Name: name, Status: model.StatusUnexpected,
			Err: fmt.Errorf("DMM GraphQL returned HTTP %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return model.Result{Name: name, Status: model.StatusNetworkErr, Err: err}
	}
	var response dmmGraphQLResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return model.Result{Name: name, Status: model.StatusErr, Err: err}
	}
	if response.Data.Client.IsForeignAccess == nil {
		return model.Result{Name: name, Status: model.StatusUnexpected,
			Err: fmt.Errorf("DMM GraphQL response does not contain isForeignAccess")}
	}
	if *response.Data.Client.IsForeignAccess {
		return model.Result{Name: name, Status: model.StatusNo}
	}
	return model.Result{Name: name, Status: model.StatusYes}
}
