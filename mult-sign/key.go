package main

import (
	"context"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"errors"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	edm "github.com/MixinNetwork/mixin/crypto/edwards25519"
	"github.com/fox-one/mint-withdraw"
)

// Key key
type Key struct {
	OutputIndex int
	CoSigners   []*CoSigner
}

// NewKey new key
func NewKey(outputIndex int, sigKey string, apiBases ...string) (*Key, error) {
	b, _ := pem.Decode([]byte(sigKey))
	if b == nil {
		return nil, errors.New("invalid sig key")
	}

	pk, err := x509.ParsePKCS1PrivateKey(b.Bytes)
	if err != nil {
		return nil, err
	}

	k := &Key{
		OutputIndex: outputIndex,
	}

	k.CoSigners = make([]*CoSigner, len(apiBases))
	for idx, api := range apiBases {
		k.CoSigners[idx] = NewCosigner(api, pk)
	}
	return k, nil
}

// VerifyOutputs verify ouputs
//	TODO
func (k Key) VerifyOutputs(t *mint.Transaction) ([]int, error) {
	return []int{k.OutputIndex}, nil
}

func (k Key) challenge(P *crypto.Key, message []byte, Rs ...*crypto.Key) [32]byte {
	var hramDigest [64]byte
	var hramDigestReduced [32]byte

	var R *crypto.Key
	for _, r := range Rs {
		if R == nil {
			R = r
		} else {
			R = crypto.KeyAddPub(R, r)
		}
	}

	h := sha512.New()
	h.Write(R[:])
	h.Write(P[:])
	h.Write(message)
	h.Sum(hramDigest[:0])
	edm.ScReduce(&hramDigestReduced, &hramDigest)
	return hramDigestReduced
}

// Sign sign transaction, only for transaction, not for mint/deposit
func (k Key) Sign(out *common.Transaction, t *mint.Transaction) (*common.VersionedTransaction, error) {
	signed := out.AsLatestVersion()
	if len(signed.Inputs) == 0 {
		return nil, nil
	}

	for idx := range out.Inputs {
		var sR *crypto.Key
		var randoms = make([]*crypto.Key, len(k.CoSigners))
		for idx, s := range k.CoSigners {
			r, err := s.RandomKey(context.Background())
			if err != nil {
				return nil, err
			}
			randoms[idx] = r

			if sR == nil {
				sR = r
			} else {
				sR = crypto.KeyAddPub(sR, r)
			}
		}

		var response *[32]byte
		for _, s := range k.CoSigners {
			resp, err := s.Sign(context.Background(), signed.Transaction, idx, randoms)
			if err != nil {
				return nil, err
			}
			if response == nil {
				response = resp
			} else {
				edm.ScAdd(response, response, resp)
			}
		}
		var sig crypto.Signature
		copy(sig[:], sR[:])
		copy(sig[32:], response[:])

		signed.Signatures = append(signed.Signatures, []crypto.Signature{sig})
	}

	return signed, nil
}
