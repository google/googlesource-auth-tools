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

// Git-credential-googlesource is a command that returns username/password for
// googlesource.com / source.developers.google.com. This command is suitable for
// a credential helper.
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/google/googlesource-auth-tools/credentials"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s PROMPT", os.Args[0])
	}

	if os.Args[1] != "get" {
		return
	}

	sc := bufio.NewScanner(os.Stdin)
	var protocol, host, path string
	for sc.Scan() {
		s := strings.TrimSpace(sc.Text())
		if s == "" {
			break
		}
		ss := strings.SplitN(s, "=", 2)
		if len(ss) != 2 {
			log.Fatalf("Cannot parse the git-credential input: %s", sc.Text())
		}
		switch ss[0] {
		case "protocol":
			protocol = ss[1]
		case "host":
			host = ss[1]
		case "path":
			path = ss[1]
		}
	}
	if err := sc.Err(); err != nil {
		log.Fatalf("Cannot parse the git-credential input: %v", err)
	}

	if host != "source.developers.google.com" && !strings.HasSuffix(host, ".googlesource.com") {
		return
	}

	u := &url.URL{}
	u.Scheme = protocol
	u.Host = host
	u.Path = path

	switch protocol {
	case "https":
		// OK
	case "http":
		gitBinary, err := credentials.FindGitBinary()
		if err != nil {
			log.Fatalf("Cannot find the git binary: %v", err)
		}
		g := gitBinary.WithURL(u)
		allowHTTP, err := g.BoolConfig(context.Background(), "google.allowHTTPForCredentialHelper")
		if err != nil {
			log.Fatalf("Cannot get a config for google.allowHTTPForCredentialHelper: %v", err)
		}
		if !allowHTTP {
			log.Fatalf("%s won't support HTTP protocol", os.Args[0])
		}
	default:
		log.Fatalf("Unknown protocol: %s", protocol)
	}

	gitBinary, err := credentials.FindGitBinary()
	if err != nil {
		log.Fatalf("Cannot find the git binary: %v", err)
	}

	token, err := credentials.MakeToken(context.Background(), gitBinary, u)
	if err != nil {
		log.Fatalf("Cannot get a token: %v", err)
	}

	fmt.Printf("protocol=%s\n", protocol)
	fmt.Printf("host=%s\n", host)
	fmt.Printf("username=git-service-account\n")
	fmt.Printf("password=%s\n", token.AccessToken)
}
