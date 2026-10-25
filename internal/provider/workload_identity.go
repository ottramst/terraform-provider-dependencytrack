package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Token exchange (RFC 8693) parameters accepted by Dependency-Track's
// POST /api/v2/oauth/token (5.2+).
const (
	tokenExchangeGrantType = "urn:ietf:params:oauth:grant-type:token-exchange"
	tokenTypeJWT           = "urn:ietf:params:oauth:token-type:jwt"
)

// oauthTokenResponse is the successful response of POST /api/v2/oauth/token.
type oauthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// oauthTokenError is the RFC 6749 error response of POST /api/v2/oauth/token.
type oauthTokenError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// exchangeWorkloadIdentityToken exchanges a workload's identity token (a JWT
// issued by a workload identity provider, e.g. a GitHub Actions OIDC token)
// for a Dependency-Track session token of serviceAccount. The session token
// is used as a bearer token and expires after expiresIn seconds.
func exchangeWorkloadIdentityToken(ctx context.Context, endpoint, providerName, serviceAccount, subjectToken string) (accessToken string, expiresIn int, err error) {
	form := url.Values{
		"grant_type":                 {tokenExchangeGrantType},
		"subject_token":              {subjectToken},
		"subject_token_type":         {tokenTypeJWT},
		"workload_identity_provider": {providerName},
		"service_account":            {serviceAccount},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(endpoint, "/")+"/api/v2/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json, application/problem+json;q=0.9")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("perform token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", 0, fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var oauthErr oauthTokenError
		if json.Unmarshal(body, &oauthErr) == nil && oauthErr.Error != "" {
			return "", 0, fmt.Errorf("token exchange refused (status %d): %s: %s", resp.StatusCode, oauthErr.Error, oauthErr.ErrorDescription)
		}
		return "", 0, &apiError{StatusCode: resp.StatusCode, Body: truncateBody(body)}
	}

	var token oauthTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return "", 0, fmt.Errorf("decode token response: %w", err)
	}
	if token.AccessToken == "" {
		return "", 0, fmt.Errorf("token response has no access_token")
	}
	return token.AccessToken, token.ExpiresIn, nil
}
