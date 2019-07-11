package main

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/httpclient"
	jsoniter "github.com/json-iterator/go"
)

// CoSigner co signer
type CoSigner struct {
	client *httpclient.Client
}

// NewCosigner new co signer
func NewCosigner(apiBase string) *CoSigner {
	return &CoSigner{
		client: httpclient.NewClient(apiBase),
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
	}
	if err := jsoniter.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	if !resp.Random.HasValue() {
		return nil, errors.New("no valid random key")
	}

	return &resp.Random, nil
}

// Sign sign
func (s CoSigner) Sign(ctx context.Context, transaction crypto.Hash, index int, hram [32]byte, random crypto.Key) (*[32]byte, error) {
	data, err := s.client.POST("/sign").
		P("transaction", transaction).
		P("index", index).
		P("hram", hex.EncodeToString(hram[:])).
		P("random", random.String()).
		Do(ctx).Bytes()
	if err != nil {
		return nil, err
	}

	var resp struct {
		Response string `json:"response"`
	}
	if err := jsoniter.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	bts, err := hex.DecodeString(resp.Response)
	if err != nil {
		return nil, err
	}

	var r [32]byte
	copy(r[:], bts[:32])
	return &r, nil
}
