package main

import (
	"crypto/rand"

	"github.com/MixinNetwork/mixin/crypto"
	edm "github.com/MixinNetwork/mixin/crypto/edwards25519"
	edk "go.dedis.ch/kyber/group/edwards25519"
)

func response(hram [32]byte, R, a, b, random *crypto.Key, signerCount int64) [32]byte {
	var s [32]byte
	privateKey := deriveGhostPrivateKey(R, a, b, 0, signerCount)
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
