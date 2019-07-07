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
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

// MakeCookies create cookies for .gitcookies.
func MakeCookies(u *url.URL, token *oauth2.Token) []*http.Cookie {
	// N.B. nscjar adds #HttpOnly_ for HttpOnly cookies, and these prevent
	// git recognize the cookies. Do not add.
	path := u.Path
	if path == "" {
		path = "/"
	}
	// The ending ".git" is redundant.
	path = strings.TrimSuffix(path, ".git")
	if u.Host == "googlesource.com" {
		// Authenticate against all *.googlesource.com.
		return []*http.Cookie{
			{
				Name:    "o",
				Value:   token.AccessToken,
				Path:    path,
				Domain:  "." + u.Host,
				Expires: token.Expiry,
				Secure:  u.Scheme == "https",
			},
		}
	} else if strings.HasSuffix(u.Host, ".googlesource.com") {
		// Authenticate against both FOO.googlesource.com and
		// FOO-review.googlesource.com. These two URLs have no
		// difference.
		h := strings.TrimSuffix(strings.TrimSuffix(u.Host, ".googlesource.com"), "-review")
		return []*http.Cookie{
			{
				Name:    "o",
				Value:   token.AccessToken,
				Path:    path,
				Domain:  h + ".googlesource.com",
				Expires: token.Expiry,
				Secure:  u.Scheme == "https",
			},
			{
				Name:    "o",
				Value:   token.AccessToken,
				Path:    path,
				Domain:  h + "-review.googlesource.com",
				Expires: token.Expiry,
				Secure:  u.Scheme == "https",
			},
		}
	}
	return []*http.Cookie{
		{
			Name:    "o",
			Value:   token.AccessToken,
			Path:    path,
			Domain:  u.Host,
			Expires: token.Expiry,
			Secure:  u.Scheme == "https",
		},
	}
}
