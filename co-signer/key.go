package main

import (
	"encoding/hex"

	"github.com/MixinNetwork/mixin/crypto"
)

// Key key
type Key struct {
	Spend crypto.Key
	View  crypto.Key

	signerCount int64
}

// NewKey create key
func NewKey(view, spend string, signerCount int64) (*Key, error) {
	decodeFunc := func(s string) (crypto.Key, error) {
		var k crypto.Key
		b, err := hex.DecodeString(s)
		if err != nil {
			return k, err
		}

		copy(k[:32], b[:32])
		return k, nil
	}
	key := &Key{
		signerCount: signerCount,
	}
	{
		k, err := decodeFunc(view)
		if err != nil {
			return nil, err
		}
		key.View = k
	}

	{
		k, err := decodeFunc(spend)
		if err != nil {
			return nil, err
		}
		key.Spend = k
	}
	return key, nil
}

// Response response
func (k Key) Response(hram [32]byte, mask, random *crypto.Key) [32]byte {
	return response(hram, mask, &k.View, &k.Spend, random, k.signerCount)
}
