// Copyright 2025 Interlynk.io
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package interlynk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ValidateInterlynkConnection chesks whether Interlynk ssytem is up and running
func ValidateInterlynkConnection(url, token string) error {
	ctx := context.Background()

	baseURL, err := genHealthzUrl(url)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	if err != nil {
		return fmt.Errorf("falied to create request for Interlynk: %w", err)
	}

	// INTERLYNK_SECURITY_TOKEN is required here
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach Interlynk at %s: %w", baseURL, err)
	}
	defer resp.Body.Close()

	// provided token is invalid
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid API token: authentication failed")
	}

	// interlynk looks to down
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Interlynk API returned unexpected status: %d", resp.StatusCode)
	}

	return nil
}

func genHealthzUrl(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s://%s/healthz", parsedURL.Scheme, parsedURL.Host), nil
}
