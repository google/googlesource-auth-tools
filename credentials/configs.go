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

package credentials

import (
	"context"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/xerrors"
)

// CredentialConfig is the configuration for credentials.
type CredentialConfig struct {
	// An account to be used. This can take one of the following values. If
	// empty, it defaults to `gcloud`.
	//
	// *   `gcloud`
	//
	//     Use the default account of `gcloud`.
	//
	// *   `application-default`
	//
	//     Use the applicaiton default credentials.
	//
	// *   Google Account emails
	//
	//     Get an access token by using `gcloud auth print-access-token
	//     EMAIL`. The account specified here must be registered in gcloud
	//     by using `gcloud auth login`
	//
	// *   Service account emails
	//     (`SERVICE_ACCOUNT@YOUR_PROJECT.iam.gserviceaccount.com`)
	//
	//     Start from the application default credentials, use IAM Service
	//     Account Credentials API to obtain the specified service account
	//     credentials. The account used for the application default service
	//     account must have `iam.serviceAccounts.getAccessToken` for the
	//     account specified here.
	//
	//     In a rare situation where you need a multi-hop delegation, you can
	//     specify a list of delegated service account emails in
	//     `ServiceAccountDelegateEmails`.
	Account string

	// OAuth2 scopes. If empty, it defaults to
	// `https://www.googleapis.com/auth/cloud-platform`.
	//
	// This config is usually not effective unless you use service account
	// emails for `Account`.
	Scopes []string

	// List of service account email addresses for multi-hop authentication.
	// See the description of `Account`.
	ServiceAccountDelegateEmails []string

	// Path to gcloud executable.
	GcloudPath string
}

// GitConfigAccessor is an interface for reading git-config.
type GitConfigAccessor interface {
	// BoolConfig returns a gitconfig config value as a boolean.
	BoolConfig(ctx context.Context, key string) (bool, error)
	// PathConfig returns a gitconfig config value as a string path.
	PathConfig(ctx context.Context, key string) (string, error)
	// StringConfig returns a gitconfig config value as a string.
	StringConfig(ctx context.Context, key string) (string, error)
	// StringListConfig returns a gitconfig config value as a string slice.
	// The value is split by comma.
	StringListConfig(ctx context.Context, key string) ([]string, error)
}

// GitBinary is a path to Git binary.
type GitBinary struct {
	// Path is a path to the Git binary.
	Path string
	// Configs are the additional Git configs specified via "-c".
	Configs []string
}

// FindGitBinary finds a git binary from the PATH.
func FindGitBinary() (GitBinary, error) {
	p, err := exec.LookPath("git")
	if err != nil {
		return GitBinary{}, xerrors.Errorf("credentials: cannot find the git binary: %v", err)
	}
	return GitBinary{Path: p}, nil
}

// ListURLs returns a list of URLs specified for "google" section.
func (g GitBinary) ListURLs(ctx context.Context) ([]*url.URL, error) {
	args := append(constructConfigArgs(g), "config", "--name-only", "--list", "--null")
	cmd := exec.CommandContext(ctx, g.Path, args...)
	cmd.Stderr = os.Stderr
	bs, err := cmd.Output()
	if err != nil {
		return nil, xerrors.Errorf("credentials: cannot get gitconfig: %v", err)
	}

	m := map[string]bool{}
	for _, s := range strings.Split(string(bs), "\000") {
		if !strings.HasPrefix(s, "google.") {
			continue
		}
		s = strings.TrimPrefix(s, "google.")
		i := strings.LastIndexByte(s, '.')
		if i > 0 {
			s = s[:i]
		}
		if !strings.Contains(s, "://") {
			continue
		}
		u, err := url.Parse(s)
		if err != nil {
			return nil, xerrors.Errorf("credentials: cannot parse the URL %s: %v", s, err)
		}
		m[u.String()] = true
	}
	urls := []*url.URL{}
	for rawURL, _ := range m {
		// The URL should be able to be re-parsed.
		u, err := url.Parse(rawURL)
		if err != nil {
			panic(err)
		}
		urls = append(urls, u)
	}
	return urls, nil
}

// ConfigFromGitConfig creates a CredentialConfig from git-config.
func (g GitBinary) CredentialConfigFromGitConfig(ctx context.Context, u *url.URL) (*CredentialConfig, error) {
	scoped := g.WithURL(u)

	c := &CredentialConfig{}

	var err error
	c.Account, err = scoped.StringConfig(ctx, "google.account")
	if err != nil {
		return nil, xerrors.Errorf("credentials: cannot get google.account config: %v", err)
	}

	c.Scopes, err = scoped.StringListConfig(ctx, "google.scopes")
	if err != nil {
		return nil, xerrors.Errorf("credentials: cannot get a list of OAuth2 scopes: %v", err)
	}

	c.ServiceAccountDelegateEmails, err = scoped.StringListConfig(ctx, "google.serviceAccountDelegateEmails")
	if err != nil {
		return nil, xerrors.Errorf("credentials: cannot get a list of service account delegates: %v", err)
	}

	c.GcloudPath, err = scoped.PathConfig(ctx, "google.gcloudPath")
	if err != nil {
		return nil, xerrors.Errorf("credentials: cannot get the gcloud path: %v", err)
	}

	return c, nil
}

// WithURL binds an URL for git-config. This makes it specify --get-urlmatch.
func (g GitBinary) WithURL(u *url.URL) GitConfigAccessor {
	return gitConfigAccessor{g, u}
}

func (g GitBinary) BoolConfig(ctx context.Context, key string) (bool, error) {
	return gitConfigAccessor{g, nil}.BoolConfig(ctx, key)
}

func (g GitBinary) PathConfig(ctx context.Context, key string) (string, error) {
	return gitConfigAccessor{g, nil}.PathConfig(ctx, key)
}

func (g GitBinary) StringConfig(ctx context.Context, key string) (string, error) {
	return gitConfigAccessor{g, nil}.StringConfig(ctx, key)
}

func (g GitBinary) StringListConfig(ctx context.Context, key string) ([]string, error) {
	return gitConfigAccessor{g, nil}.StringListConfig(ctx, key)
}

type gitConfigAccessor struct {
	gitBinary GitBinary
	u         *url.URL
}

func (g gitConfigAccessor) get(ctx context.Context, ty, key string) (string, error) {
	args := append(constructConfigArgs(g.gitBinary), "config", ty)
	if g.u != nil {
		args = append(args, "--get-urlmatch", key, g.u.String())
	} else {
		args = append(args, key)
	}
	cmd := exec.CommandContext(ctx, g.gitBinary.Path, args...)
	cmd.Stderr = os.Stderr
	bs, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if ee.ExitCode() == 1 {
				// The key doesn't exist in the config.
				return "", nil
			}
		}
		return "", xerrors.Errorf("credentials: cannot get gitconfig: %v", err)
	}
	return strings.TrimSpace(string(bs)), nil
}

func (g gitConfigAccessor) BoolConfig(ctx context.Context, key string) (bool, error) {
	v, err := g.get(ctx, "--bool", key)
	if err != nil {
		return false, err
	}
	return v != "false", nil
}

func (g gitConfigAccessor) PathConfig(ctx context.Context, key string) (string, error) {
	return g.get(ctx, "--path", key)
}

func (g gitConfigAccessor) StringConfig(ctx context.Context, key string) (string, error) {
	return g.get(ctx, "--no-type", key)
}

func (g gitConfigAccessor) StringListConfig(ctx context.Context, key string) ([]string, error) {
	v, err := g.get(ctx, "--no-type", key)
	if err != nil {
		return nil, err
	}
	if v == "" {
		return nil, nil
	}
	ss := []string{}
	for _, s := range strings.Split(v, ",") {
		ss = append(ss, strings.TrimSpace(s))
	}
	return ss, nil
}

func constructConfigArgs(g GitBinary) []string {
	args := []string{}
	for _, c := range g.Configs {
		args = append(args, "-c", c)
	}
	return args
}
