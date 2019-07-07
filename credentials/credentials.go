// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package credentials provides OAuth2 TokenSource built based on the gitconfig
// configs.
package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/xerrors"
	"google.golang.org/api/iamcredentials/v1"
	"google.golang.org/api/option"
)

const (
	scopeCloudPlatform = "https://www.googleapis.com/auth/cloud-platform"

	accountApplicationDefault = "application-default"
	accountGcloud             = "gcloud"
)

// MakeToken creates a token for the given URL.
func MakeToken(ctx context.Context, u *url.URL) (*oauth2.Token, error) {
	c, err := ConfigFromGitConfig(ctx, u)
	if err != nil {
		return nil, xerrors.Errorf("credentials: cannot get configs: %v", err)
	}
	ts, err := TokenSourceFromConfig(ctx, c)
	if err != nil {
		return nil, xerrors.Errorf("credentials: cannot get a TokenSource: %v", err)
	}
	token, err := ts.Token()
	if err != nil {
		return nil, xerrors.Errorf("credentials: cannot get a token: %v", err)
	}
	return token, nil
}

// TokenSourceFromConfig returns a TokenSource configured based on gitconfig.
func TokenSourceFromConfig(ctx context.Context, c *CredentialConfig) (oauth2.TokenSource, error) {
	account := c.Account
	if account == "" {
		c.Account = accountGcloud
	}
	scopes := c.Scopes
	if len(scopes) == 0 {
		scopes = []string{scopeCloudPlatform}
	}

	switch {
	case account == accountGcloud:
		return newGcloudTokenSource(ctx, c, "")

	case account == accountApplicationDefault:
		// Use the application default credentials.
		ts, err := google.DefaultTokenSource(ctx, scopes...)
		if err != nil {
			return nil, xerrors.Errorf("credentials: cannot get the application default credentials: %v", err)
		}
		return ts, nil

	case strings.HasSuffix(account, ".gserviceaccount.com"):
		//Use IAM credentials API
		ts, err := google.DefaultTokenSource(ctx, scopeCloudPlatform)
		if err != nil {
			return nil, xerrors.Errorf("credentials: cannot get the application default credentials: %v", err)
		}

		svc, err := iamcredentials.NewService(ctx, option.WithTokenSource(ts))
		if err != nil {
			return nil, xerrors.Errorf("credentials: cannot create an IAM Service Account Credentials API client: %v", err)
		}

		ds := []string{}
		for _, d := range c.ServiceAccountDelegateEmails {
			ds = append(ds, fmt.Sprintf("projects/-/serviceAccounts/%s", d))
		}
		return oauth2.ReuseTokenSource(nil, &iamCredentialsTokenSource{
			name:           fmt.Sprintf("projects/-/serviceAccounts/%s", account),
			delegates:      ds,
			scopes:         scopes,
			iamCredService: iamcredentials.NewProjectsServiceAccountsService(svc),
		}), nil

	default:
		return newGcloudTokenSource(ctx, c, account)
	}
}

func newGcloudTokenSource(ctx context.Context, c *CredentialConfig, name string) (oauth2.TokenSource, error) {
	gcloudPath := c.GcloudPath
	var err error
	if gcloudPath == "" {
		gcloudPath, err = exec.LookPath("gcloud")
		if err != nil {
			return nil, xerrors.Errorf("credentials: cannot find the gcloud binary: %v", err)
		}
	}
	gcloudPath, err = filepath.Abs(gcloudPath)
	if err != nil {
		return nil, xerrors.Errorf("credentials: cannot get an absolute path to gcloud: %v", err)
	}
	return oauth2.ReuseTokenSource(nil, &gcloudTokenSource{
		name:       name,
		gcloudPath: gcloudPath,
	}), nil
}

type gcloudTokenSource struct {
	name       string
	gcloudPath string
}

func (s *gcloudTokenSource) Token() (*oauth2.Token, error) {
	ss := []string{
		"--format=json", "auth", "print-access-token",
	}
	if s.name != "" {
		ss = append(ss, s.name)
	}
	cmd := exec.CommandContext(context.Background(), s.gcloudPath, ss...)
	cmd.Stderr = os.Stderr
	bs, err := cmd.Output()
	if err != nil {
		return nil, xerrors.Errorf("credentials: failed to run gcloud: %v", err)
	}

	cred := &gcloudCredential{}
	if err := json.Unmarshal(bs, cred); err != nil {
		return nil, xerrors.Errorf("credentials: failed to parse gcloud print-access-token result: %v", err)
	}
	if cred.AccessToken == "" || cred.TokenExpiry.DateTime == "" {
		return nil, xerrors.Errorf("credentials: incomplete gcloud print-access-token result: %v", cred)
	}
	expiry, err := time.Parse("2006-01-02 15:04:05.000000", cred.TokenExpiry.DateTime)
	if err != nil {
		return nil, xerrors.Errorf("credentials: cannot parse TokenExpiry: %v", err)
	}
	return &oauth2.Token{
		AccessToken: cred.AccessToken,
		Expiry:      expiry,
	}, nil
}

type gcloudCredential struct {
	AccessToken string `json:"access_token"`
	TokenExpiry struct {
		DateTime string `json:"datetime"`
	} `json:"token_expiry"`
}

type iamCredentialsTokenSource struct {
	name           string
	delegates      []string
	scopes         []string
	iamCredService *iamcredentials.ProjectsServiceAccountsService
}

func (s *iamCredentialsTokenSource) Token() (*oauth2.Token, error) {
	resp, err := s.iamCredService.GenerateAccessToken(s.name, &iamcredentials.GenerateAccessTokenRequest{
		Delegates: s.delegates,
		Scope:     s.scopes,
	}).Context(context.Background()).Do()
	if err != nil {
		return nil, xerrors.Errorf("credentials: cannot obtain a credential: %v", err)
	}
	expiry, err := time.Parse(time.RFC3339Nano, resp.ExpireTime)
	if err != nil {
		return nil, xerrors.Errorf("credentials: cannot parse ExpireTime: %v", err)
	}
	return &oauth2.Token{
		AccessToken: resp.AccessToken,
		Expiry:      expiry,
	}, nil
}
