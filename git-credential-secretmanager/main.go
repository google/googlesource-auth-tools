package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/googleapis/gax-go"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

var (
	version = flag.String("version", "", "Secret Manager version (e.g. projects/my-project/secrets/my-secret/versions/latest)")
)

func main() {
	flag.Parse()

	v := *version
	if v == "" {
		v = os.Getenv("GIT_SECRET_MANAGER_VERSION")
	}
	if v == "" {
		fmt.Fprintf(os.Stderr, "%s: cannot determine Secret Manager version, --version or ${GIT_SECRET_MANAGER_VERSION} not specified\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	if os.Args[len(os.Args)-1] != "get" {
		return
	}

	fmt.Fprintln(os.Stderr, v)

	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		log.Fatalf("failed to create secretmanager client: %v", err)
	}

	if err := generateCreds(ctx, os.Stdin, os.Stdout, client, v); err != nil {
		log.Fatal(err)
	}
}

type secretGetter interface {
	AccessSecretVersion(context.Context, *secretmanagerpb.AccessSecretVersionRequest, ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error)
}

func generateCreds(ctx context.Context, r io.Reader, w io.Writer, client secretGetter, version string) error {
	// Read in Git credential config - https://git-scm.com/docs/git-credential#IOFMT
	cred, err := read(os.Stdin)
	if err != nil {
		return fmt.Errorf("error reading stdin: %v", err)
	}

	// Get secret from SecretManager
	result, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: version,
	})
	if err != nil {
		return fmt.Errorf("failed to access secret version: %v", err)
	}

	// Write secret back out to Git credential.
	cred.password = string(result.Payload.Data)
	cred.write(w)

	return nil
}

type credential struct {
	protocol string
	host     string
	path     string
	username string
	password string
	url      string
}

func read(r io.Reader) (credential, error) {
	var c credential
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		s := strings.SplitN(scanner.Text(), "=", 2)
		switch s[0] {
		case "protocol":
			c.protocol = s[1]
		case "host":
			c.host = s[1]
		case "path":
			c.path = s[1]
		case "username":
			c.username = s[1]
		case "password":
			c.password = s[1]
		case "url":
			c.url = s[1]
		}
	}
	return c, scanner.Err()
}

func (c credential) write(w io.Writer) {
	printIfSet(w, "protocol", c.protocol)
	printIfSet(w, "host", c.host)
	printIfSet(w, "path", c.path)
	printIfSet(w, "username", c.username)
	printIfSet(w, "password", c.password)
	printIfSet(w, "url", c.url)
}

func printIfSet(w io.Writer, k, v string) {
	if v != "" {
		fmt.Fprintf(w, "%s=%s\n", k, v)
	}
}
