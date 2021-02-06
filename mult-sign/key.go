package main

import (
	"context"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"errors"

	edm "filippo.io/edwards25519"
	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mint-withdraw"
	log "github.com/sirupsen/logrus"
)

// Key key
type Key struct {
	SpendPub *crypto.Key
	View     *crypto.Key

	CoSigners []*CoSigner
}

// NewKey new key
func NewKey(spendPub, viewPriv string, sigKey string, apiBases ...string) (*Key, error) {
	b, _ := pem.Decode([]byte(sigKey))
	if b == nil {
		return nil, errors.New("invalid sig key")
	}

	pk, err := x509.ParsePKCS1PrivateKey(b.Bytes)
	if err != nil {
		return nil, err
	}

	k := &Key{}

	k.SpendPub, err = decodeKey(spendPub)
	if err != nil {
		return nil, err
	}

	if viewPriv != "" {
		k.View, err = decodeKey(viewPriv)
		if err != nil {
			return nil, err
		}
	} else {
		key := k.SpendPub.DeterministicHashDerive()
		k.View = &key
	}

	k.CoSigners = make([]*CoSigner, len(apiBases))
	for idx, api := range apiBases {
		k.CoSigners[idx] = NewCosigner(api, pk)
	}
	return k, nil
}

// VerifyOutputs verify ouputs
func (k Key) VerifyOutputs(t *mint.Transaction) ([]int, error) {
	var outputs = make([]int, 0, len(t.Outputs))
	for idx, o := range t.Outputs {
		for _, key := range o.Keys {
			if crypto.ViewGhostOutputKey(&key, k.View, &o.Mask, uint64(idx)).String() == k.SpendPub.String() {
				outputs = append(outputs, idx)
				break
			}
		}
	}
	return outputs, nil
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
		log.Println("sign with no input")
		return nil, nil
	}

	for idx := range out.Inputs {
		var sR *crypto.Key
		var randoms = make([]*crypto.Key, len(k.CoSigners))
		for idx, s := range k.CoSigners {
			r, err := s.RandomKey(context.Background())
			log.Println("random key", s.apiBase, r, err)
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
			log.Println("sign", s.apiBase, resp, err)
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
