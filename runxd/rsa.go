// +build ignore

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"
)

func main() {
	k, err := rsa.GenerateKey(rand.Reader, 768)
	if err != nil {
		log.Fatal(err)
	}
	var b pem.Block
	b.Type = "RSA PRIVATE KEY"
	b.Bytes = x509.MarshalPKCS1PrivateKey(k)
	pem.Encode(os.Stdout, &b)
}
