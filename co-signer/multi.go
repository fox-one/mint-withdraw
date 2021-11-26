package main

import (
	"crypto/rand"
	"crypto/sha512"

	edm "filippo.io/edwards25519"
	"github.com/MixinNetwork/mixin/crypto"
	edk "go.dedis.ch/kyber/v3/group/edwards25519"
)

func challenge(P *crypto.Key, message []byte, Rs ...*crypto.Key) [32]byte {
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

func response(hram [32]byte, R, a, b, random *crypto.Key, index uint64, signerCount int64) [32]byte {
	var s [32]byte
	privateKey := deriveGhostPrivateKey(R, a, b, index, signerCount)
	messageDigestReduced := [32]byte(*random)
	expandedSecretKey := [32]byte(*privateKey)
	edm.ScMulAdd(&s, &hram, &expandedSecretKey, &messageDigestReduced)
	return s
}

func deriveGhostPrivateKey(R, a, b *crypto.Key, outputIndex uint64, signerCount int64) *crypto.Key {
	scalar := crypto.KeyMultPubPriv(R, a).MultScalar(outputIndex).HashScalar()
	scalar = computeViewShare(scalar, signerCount)
	tmp := [32]byte(*b)
	edm.ScAdd(&tmp, &tmp, scalar)
	key := crypto.Key(tmp)
	return &key
}

func computeViewShare(a *[32]byte, count int64) *[32]byte {
	suite := edk.NewBlakeSHA256Ed25519()
	s := suite.Scalar().SetBytes(a[:])
	c := suite.Scalar().SetInt64(count)
	r := suite.Scalar().Div(s, c)

	var k [32]byte
	b, _ := r.MarshalBinary()
	copy(k[:], b)
	return &k
}

func randomKey() crypto.Key {
	seed := make([]byte, 64)
	rand.Read(seed)
	return crypto.NewKeyFromSeed(seed)
}
