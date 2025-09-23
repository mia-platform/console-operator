// Copyright 2016-2025 Mia-Platform
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package consoleclient

import (
	"context"
	"net/http"
	"time"
)

type Client struct {
	baseURL      string
	clientID     string
	clientSecret string
	httpClient   *http.Client
	ctx          context.Context
	token        string
	tokenExpiry  time.Time
}

type ClientConfig struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	Timeout      time.Duration
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope,omitempty"`
}

type ConsoleCompany struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// Add other fields as needed based on your API response
}
