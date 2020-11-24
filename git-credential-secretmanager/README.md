# git-credential-secretmanager

A Git credential helper for accessing GCP Secret Manager secrets.

## Usage

### Flag based configuration

```sh
$ git config --global credential.https://github.com.helper "secretmanager --version=<secret manager version>"
$ git config --global credential.https://github.com.username <username>
```

### Environment variable based configuration

```sh
$ git config --global credential.https://github.com.helper secretmanager
$ git config --global credential.https://github.com.username <username>
$ export GIT_SECRET_MANAGER_VERSION="<secret manager version>"
```


Where `<username>` is the remote username you want to use and
`<secret manager version>` is a
[GCP Secret Manager Version ID](https://cloud.google.com/secret-manager/docs/creating-and-accessing-secrets#access).
