# Git auth helpers for googlesource.com / source.developers.google.com

This is a collection of tools / libraries for making a request to
googlesource.com and source.developers.google.com with an OAuth2 tokens.

This comes with three tools and a library for programmatic access.

*   git-credential-googlesource: A gitcredentials helper
*   googlesource-askpass: A `GIT_ASKPASS` helper
*   googlesource-cookieauth: A tool to create a gitcookie file
*   credentials: A library for creating a TokenSource

These tools work without any configuration by default as long as you have gcloud
installed. For choosing which one to use, see the usage guide below.

This is not an official Google product (i.e. a 20% project).

## Install

Install like other Go tools.

## Usage

There are two things to consider for setting this up.

1.  Which account to use and how to obtain the credentials
2.  How to run these auth helpers

### Which account to use and how to obtain the credentials

*   Use on a personal machine

    If you use these tools on your workstation, which means that you use these
    tools for non-production jobs, you can just use your Google Account. Install
    [Google Cloud SDK](https://cloud.google.com/sdk/), and set up your account
    with `gcloud auth login`. These tools automatically use the default account.
    If you have multiple accounts, and you want to use different accounts for
    different repositories, see the configurations section below.

*   Use on GCE

    If you use these tools on GCE, you must use a service account. You can
    specify a service account for each GCE instance when you start one, and the
    machine can get a credential of that account from a special IP address that
    is internal to GCE. Use `https://www.googleapis.com/auth/cloud-platform` as
    the OAuth2 scope.  If you have gcloud installed in the machine, you don't
    need further configuration. If you do not want to install gcloud, you can
    specify `application-default` for `google.account` in git-config. See the
    configurations section below.

*   Use on an on-premise servers

    If you use these tools on on-premise machines, you must use a service
    account as well. In Google Cloud Platform Console, you can create a service
    account and you can download a credential JSON file for the account. You can
    distribute this credential file to your on-premise machines and specify the
    file path in `GOOGLE_APPLICATION_CREDENTIALS`. Specify `application-default`
    for `google.account` in your .gitconfig. See [Application Default
    Credentials](https://cloud.google.com/docs/authentication/production) for
    details.

### How to run these auth helpers

*   Run `googlesource-cookieauth` as a cron job

    If you don't mind running a small background cron job on your machine once
    per hour, running `googlesource-cookieauth` as a cron job will be most
    convenient and reliable option.

    Setting up a cron job depends on your operating system. Explaining how to
    set up one is beyond this help. For Linux, you might be able to use systemd
    or crontab. For Mac OS X, you might be able to use launchd or crontab. For
    Windows, you might be able to use Task Scheduler.

*   Run `googlesource-cookieauth` right before running Git commands

    The OAuth2 tokens written by `googlesource-cookieauth` are usually valid for
    an hour. If you need to access Git repositories in a CI/CD pipeline and you
    cannot modify the worker image for running a cron job, you can run this
    command once at the beginning.


*   Use `git-credential-googlesource`

    If you install `git-credential-googlesource` to the $PATH, you can specify
    `googlesource` to `credential.helper` in git-config. You can specify the
    full path as well.

    Due to the nature of git-credential mechanism, this doesn't work well for
    googlesource.com repositories. This is because Git invokes credential
    helpers only when it sees 401 Unauthorized. For public repositories, you can
    always access them and Git won't use the credential. You can change the
    repository URL path with `/a/` for force authentication. For example, you
    can use `https://gerrit.googlesource.com/a/gerrit` instead of
    `https://gerrit.googlesource.com/gerrit`. The googlesource.com server
    returns 401 Unauthorized if the request is not authenticated.

    In Mac OS X, the operating system specifies `git-credential-osxkeychain` as
    a system default credential helper. This credential helper caches the OAuth2
    access token returned by `git-credential-googlesource`. Since OAuth2 access
    tokens are valid only for a short period, the cached credential will become
    invalid quickly. This causes many confusions and it's better to disable it
    if you use `git-credential-googlesource`. (A patch for disabling cache is
    sent to git upstream
    https://public-inbox.org/git/20190707055132.103736-1-masayasuzuki@google.com/T/#u)

*   Use `googlesource-askpass`

    You can specify the path to `googlesource-askpass` to `GIT_ASKPASS`
    environment variable or `core.askPass` in git-config.

    The same restriction applies for `googlesource-askpass`, and this doesn't
    work well for googlesource.com repositories.

## Configurations

Most of the configurations can be done via git-config. Consult the git manual
pages on how to configure the options.

*   `google.account`

    An account to be used. This can take one of the following values. If empty,
    it defaults to `gcloud`.

    *   `gcloud`

        Use the default account of `gcloud`.

    *   `application-default`

        Use the applicaiton default credentials.

    *   Google Account emails

        Get an access token by using `gcloud auth print-access-token EMAIL`. The
        account specified here must be registered in gcloud by using `gcloud
        auth login`

    *   Service account emails
        (`SERVICE_ACCOUNT@YOUR_PROJECT.iam.gserviceaccount.com`)

        Start from the application default credentials, use IAM Service Account
        Credentials API to obtain the specified service account credentials. The
        account used for the application default service account must have
        `iam.serviceAccounts.getAccessToken` for the account specified here.

        In a rare situation where you need a multi-hop delegation, you can
        specify a list of delegated service account emails in
        `google.serviceAccountDelegateEmails`.

*   `google.scopes`

    Comma separated values of OAuth2 scopes. If empty, it defaults to
    `https://www.googleapis.com/auth/cloud-platform`.

    This config is usually not effective unless you use service account emails
    for `google.account`.

*   `google.allowHTTPForCredentialHelper`

    A boolean value that is used only for `git-credential-googlesource`. If
    true, it allows returning a credential for HTTP URLs. Usually this is not
    needed, but this comes handy if you have an HTTP proxy.

*   `google.cookieFile`

    A file path to a cookie file. `googlesource-cookieauth` writes Netscape
    cookies to this file. If you specify "-", it writes to stdout. If empty, it
    defaults to `$HOME/.git-credential-cache/googlesource-cookieauth-cookie`.

*   `google.gcloudPath`

    A file path to `gcloud`. If empty, it defaults to the one in the $PATH.

All configurations above, except `google.cookieFile`, can be scoped to a URL by
using `google.<url>.*` syntax. For example, if you want to use your Gmail
address by default, and use your chromium.org account only for
chromium.googlesource.com, you can write the following .gitconfig.

```
[google]
  account = johndoe@gmail.com
[google "https://chromium.googlesource.com"]
  account = johndoe@chromium.org
```

For `googlesource-cookieauth`, you can specify `google.cookieFile` via a command
line flag, too. Specify a file path via `--output`. The commandline flag takes a
precedence over git-config.
