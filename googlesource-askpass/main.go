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

// Googlesource-askpass is a command that returns username/password for
// googlesource.com / source.developers.google.com. This command is suitable for
// GIT_ASKPASS.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/googlesource-auth-tools/credentials"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s PROMPT", os.Args[0])
	}

	prompt := strings.ToLower(os.Args[1])
	if strings.Contains(prompt, "username") {
		fmt.Print("git-service-account")
		return
	} else if strings.Contains(prompt, "password") {
		gitBinary, err := credentials.FindGitBinary()
		if err != nil {
			log.Fatalf("Cannot find the git binary: %v", err)
		}
		token, err := credentials.MakeToken(context.Background(), gitBinary, nil)
		if err != nil {
			log.Fatalf("Cannot get a token: %v", err)
		}
		fmt.Print(token.AccessToken)
		return
	}
	log.Fatalf("Unrecognized prompt")
}
