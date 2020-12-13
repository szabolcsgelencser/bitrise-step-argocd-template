package gitops

import (
	"context"
	"crypto/rsa"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestSSHKey(t *testing.T) {
	ctx := context.Background()
	wantGithubKeyID := time.Now().Second() + 1 // random from 1...60

	// Initialize mock Github client.
	var gotAuthorizedKey string
	var deletedKeyID int
	gh := &githuberMock{
		AddKeyFunc: func(_ context.Context, pub []byte) (int64, error) {
			gotAuthorizedKey = string(pub)
			return int64(wantGithubKeyID), nil
		},
		DeleteKeyFunc: func(_ context.Context, id int64) error {
			deletedKeyID = int(id)
			return nil
		},
	}

	// Create new SSH key.
	sshKey, err := NewSSHKey(ctx, gh)
	require.NoError(t, err, "newSSHKey")

	// Assert local private key and Github deploy key are a valid pair.
	privatePath := sshKey.privateKeyPath()
	gotPrivateKeyBytes, err := ioutil.ReadFile(privatePath)
	require.NoError(t, err, "read contents of %q", privatePath)
	gotPrivateKey, err := ssh.ParseRawPrivateKey(gotPrivateKeyBytes)
	require.NoError(t, err, "ssh.ParseRawPrivateKey")
	gotRSAPrivateKey, ok := gotPrivateKey.(*rsa.PrivateKey)
	require.True(t, ok, "gotPrivateKey.(*rsa.PrivateKey)")

	wantPublicKey := &gotRSAPrivateKey.PublicKey
	gotPublicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(gotAuthorizedKey))
	require.NoError(t, err, "ssh.ParseAuthorizedKey")

	require.EqualValues(t, wantPublicKey, gotPublicKey, "deployed key matches local private key")

	// Assert close deletes deploy key from Github.
	assert.Equal(t, 0, deletedKeyID, "should not be deleted before close call")
	sshKey.close(ctx)
	assert.Equal(t, wantGithubKeyID, deletedKeyID, "deleted key ID")
}
