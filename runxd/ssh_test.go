package main

import (
	"code.google.com/p/go.crypto/ssh"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"os/user"
	"testing"
)

func TestSSHD(t *testing.T) {
	block, _ := pem.Decode([]byte(testClientPrivateKey))
	rsakey, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
	pub, _ := ssh.NewPublicKey(&rsakey.PublicKey)
	cmd, c, err := startSSHD(ssh.MarshalAuthorizedKey(pub))
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Wait()
	defer cmd.Process.Kill()
	u, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	_ = u
	config := &ssh.ClientConfig{
		User: u.Username,
		Auth: []ssh.ClientAuth{ssh.ClientAuthKeyring(&keyring{rsakey})},
	}
	client, err := ssh.Client(c, config)
	if err != nil {
		t.Fatal(err)
	}
	sess, err := client.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	out, err := sess.Output("echo hello")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "hello\n" {
		t.Fatalf("out = %q want %q", string(out), "hello\n")
	}
}

type keyring struct {
	key *rsa.PrivateKey
}

func (k *keyring) Key(i int) (ssh.PublicKey, error) {
	if i != 0 {
		return nil, nil
	}
	return ssh.NewPublicKey(&k.key.PublicKey)
}

func (k *keyring) Sign(i int, rand io.Reader, data []byte) (sig []byte, err error) {
	hashFunc := crypto.SHA1
	h := hashFunc.New()
	h.Write(data)
	digest := h.Sum(nil)
	return rsa.SignPKCS1v15(rand, k.key, hashFunc, digest)
}

const testClientPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIBygIBAAJhAOKIvNBvwExqOLEYpiD+hwQtZWC2koMEOqhIJ9Mjnvqs2tLI2rRd
Cg1uLD8BTQ49EVoCJJJDsTHqkYq6JQZSQHjbbIpMN/hiwXVVOibgDo27qhabfogV
uHWPKOJINzmgQwIDAQABAmBNzsV7mkakeH+MZHj7MDFTv/voIg1krtku38m9/agn
VaO7bn2gIazIPCU6Zsn+r/5WNt7iMslUXZBBcdcEkkSIxT4eAj1cGRSiuZd/xHC+
JnAd9KL+OgyCpN4N/T+CiRECMQD4k+ooieRDLEa0IFwoE8lR/GwwjQ1Arg8W+Me5
nPwOGMrm9MNL7w7KQXsw7NVQmDcCMQDpTFMxgLb+LWWTUZAFyKGmuc1umaqzz7XC
h4wLo6TEkHHpnAe9ptkEbVrd7FhKmlUCMAEqGDe2ZaZW58HiQOxDI3dJ2mvjzUMX
TaTK54ycCqY6QYERdnS9mvEhm2UgRuOIwwIxAIGv0wNOqOrMs41cJrKAYBP9b0xP
EcxY55IWpWwG8N3v6dLR0J/Fcxf57iw1aLM37QIwaP8s8uk6BKCEvSpATD6hrFIB
BRJH2KGuCeun/roNPqQsNvqGdG6nhegWrhtkXrAl
-----END RSA PRIVATE KEY-----`
