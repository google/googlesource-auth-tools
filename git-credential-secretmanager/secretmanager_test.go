package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/googleapis/gax-go"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

func TestCredentialRender(t *testing.T) {
	for _, tc := range []struct {
		name       string
		credential string
	}{
		{
			name: "complete",
			// Note: this test is somewhat brittle since it depends on field ordering.
			// In practice, the ordering does not matter.
			credential: `protocol=https
host=example.com
path=/asdf
username=foo
password=hunter2
url=example.com
`,
		},
		{
			name:       "partial",
			credential: "protocol=https\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := read(bytes.NewBufferString(tc.credential))
			if err != nil {
				t.Fatalf("read: %v", err)
			}

			out := new(bytes.Buffer)
			c.write(out)

			if tc.credential != out.String() {
				t.Errorf("\nWant:\n%s\nGot:\n%s", tc.credential, out.String())
			}
		})
	}
}

type fakeSecretManager struct{}

func (s *fakeSecretManager) AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	return &secretmanagerpb.AccessSecretVersionResponse{
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte("hunter2"),
		},
	}, nil
}

func TestGenerateCreds(t *testing.T) {
	ctx := context.Background()
	client := &fakeSecretManager{}

	out := new(bytes.Buffer)
	if err := generateCreds(ctx, &bytes.Buffer{}, out, client, "foo"); err != nil {
		t.Fatalf("generateCreds: %v", err)
	}

	want := "password=hunter2\n"
	if want != out.String() {
		t.Errorf("want: %s, got: %s", want, out.String())
	}
}
