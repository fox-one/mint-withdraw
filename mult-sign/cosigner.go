package main

import (
	"context"
	cCrypto "crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/httpclient"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

// CoSigner co signer
type CoSigner struct {
	client *httpclient.Client

	sigKey *rsa.PrivateKey
}

// NewCosigner new co signer
func NewCosigner(apiBase string, sigKey *rsa.PrivateKey) *CoSigner {
	return &CoSigner{
		client: httpclient.NewClient(apiBase),
		sigKey: sigKey,
	}
}

// Auth add auth token
func (s CoSigner) Auth(req *httpclient.Request, method, uri string, body []byte) {
	h := sha256.New()
	h.Write(append([]byte(method+uri), body...))
	digest := h.Sum(nil)

	bts, err := rsa.SignPKCS1v15(rand.Reader, s.sigKey, cCrypto.SHA256, digest)
	if err == nil {
		req.H("Authorization", base64.StdEncoding.EncodeToString(bts))
	}
}

// RandomKey generate random key
func (s CoSigner) RandomKey(ctx context.Context) (*crypto.Key, error) {
	data, err := s.client.POST("/random").Do(ctx).Bytes()
	if err != nil {
		return nil, err
	}

	var resp struct {
		Random crypto.Key `json:"random"`

		Code    int    `json:"code"`
		Message string `json:"msg"`
	}
	if err := jsoniter.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	if resp.Code > 0 {
		return nil, fmt.Errorf("code: %d; msg: %s", resp.Code, resp.Message)
	}

	if !resp.Random.HasValue() {
		return nil, errors.New("no valid random key")
	}

	return &resp.Random, nil
}

// Sign sign
func (s CoSigner) Sign(ctx context.Context, transaction common.Transaction, index int, randoms []*crypto.Key) (*[32]byte, error) {
	data, err := s.client.POST("/sign").
		P("transaction", transaction).
		P("index", index).
		P("randoms", randoms).
		Auth(s).Do(ctx).Bytes()

	if err != nil {
		return nil, err
	}

	var resp struct {
		Response string `json:"response"`

		Code    int    `json:"code"`
		Message string `json:"msg"`
	}
	log.Println(string(data))
	if err := jsoniter.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	if resp.Code > 0 {
		return nil, fmt.Errorf("code: %d; msg: %s", resp.Code, resp.Message)
	}

	bts, err := hex.DecodeString(resp.Response)
	if err != nil {
		return nil, err
	}

	var r [32]byte
	copy(r[:], bts[:32])
	return &r, nil
}
