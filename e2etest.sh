#!/bin/bash
# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Minimum end-to-end test
#
# Usage: ./e2etest.sh USER_EMAIL SERVICE_ACCOUNT_EMAIL
#
# USER_EMAIL should be an account registered in gcloud. SERVICE_ACCOUNT_EMAIL
# should be a service account that is accessible from the application default
# credential.

tmpdir=$(mktemp -d)
atexit() {
  rm -rf $tmpdir
}
trap atexit EXIT

gcloud_user=$1
service_account_user=$2

cd /
export GIT_CONFIG_NOSYSTEM=1
export GIT_TERMINAL_PROMPT=0
orig_home=$HOME
export HOME=$tmpdir
export GIT_CONFIG=$HOME/.gitconfig
touch $GIT_CONFIG
ln -s $orig_home/.config $HOME/.config

# ------------------------------------------------------------------------------
# Auth config tests
# ------------------------------------------------------------------------------

# google.account = <GOOGLE_ACCOUNT>
cat >$HOME/.gitconfig <<EOF
[google]
  account = $gcloud_user
EOF
googlesource-cookieauth -o $HOME/.gitcookies || exit 1
git -c http.cookieFile=$HOME/.gitcookies ls-remote https://code.googlesource.com/a/git >/dev/null || exit 1

# google.account = application-default
cat >$HOME/.gitconfig <<EOF
[google]
  account = application-default
EOF
googlesource-cookieauth -o $HOME/.gitcookies || exit 1
git -c http.cookieFile=$HOME/.gitcookies ls-remote https://code.googlesource.com/a/git >/dev/null || exit 1

# google.account = <SERVICE_ACCOUNT>
cat >$HOME/.gitconfig <<EOF
[google]
  account = $service_account_user
EOF
googlesource-cookieauth -o $HOME/.gitcookies || exit 1
git -c http.cookieFile=$HOME/.gitcookies ls-remote https://code.googlesource.com/a/git >/dev/null || exit 1

# ------------------------------------------------------------------------------
# Auth helper tests
# ------------------------------------------------------------------------------

# Sanity check. /a/ URLs cannot be accessed without a credential.
if git ls-remote https://code.googlesource.com/a/git >/dev/null 2>/dev/null; then
  exit 1
fi

# git-credential-googlesource
git -c credential.helper=googlesource ls-remote https://code.googlesource.com/a/git >/dev/null || exit 1

# googlesource-askpass
GIT_ASKPASS=googlesource-askpass git ls-remote https://code.googlesource.com/a/git >/dev/null || exit 1

# googlesource-cookieauth
googlesource-cookieauth -o $HOME/.gitcookies || exit 1
git -c http.cookieFile=$HOME/.gitcookies ls-remote https://code.googlesource.com/a/git >/dev/null || exit 1
