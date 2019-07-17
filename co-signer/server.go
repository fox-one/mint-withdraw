package main

import (
	cCrypto "crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/gin-contrib/gin_helper"
	"github.com/fox-one/mint-withdraw"
	"github.com/gin-gonic/gin"
)

type serverImp struct {
	store  *Store
	key    *Key
	sigKey *rsa.PublicKey
}

func newServerImp(view, spend, sigKey string, cosignerCount int64) (*serverImp, error) {
	k, err := NewKey(view, spend, cosignerCount)
	if err != nil {
		return nil, err
	}

	b, _ := pem.Decode([]byte(sigKey))
	if b == nil {
		return nil, errors.New("invalid sig key")
	}

	pk, err := x509.ParsePKIXPublicKey(b.Bytes)
	if err != nil {
		return nil, err
	}

	return &serverImp{
		store:  NewStore(),
		key:    k,
		sigKey: pk.(*rsa.PublicKey),
	}, nil
}

func (imp *serverImp) extractBody(c *gin.Context) (body []byte, err error) {
	if cb, ok := c.Get(gin.BodyBytesKey); ok {
		if cbb, ok := cb.([]byte); ok {
			body = cbb
		}
	}

	if body == nil {
		body, err = ioutil.ReadAll(c.Request.Body)
		if err == nil {
			c.Set(gin.BodyBytesKey, body)
		}
	}

	return
}

func (imp *serverImp) sigRequired(c *gin.Context) {
	method := c.Request.Method
	uri := c.Request.URL.String()
	body, _ := imp.extractBody(c)

	h := sha256.New()
	h.Write(append([]byte(method+uri), body...))
	digest := h.Sum(nil)

	sig, err := base64.StdEncoding.DecodeString(c.GetHeader("Authorization"))
	if err != nil {
		gin_helper.FailError(c, err)
		return
	}

	if err := rsa.VerifyPKCS1v15(imp.sigKey, cCrypto.SHA256, digest, sig); err != nil {
		gin_helper.FailError(c, err)
	}
}

func (imp *serverImp) random(c *gin.Context) {
	r := randomKey()
	R := r.Public()
	key := fmt.Sprintf("random_%s", R.String())
	imp.store.WriteProperty(c, key, r.String())
	gin_helper.OK(c, "random", R.String())
}

func (imp *serverImp) sign(c *gin.Context) {
	var input struct {
		Transaction crypto.Hash `json:"transaction"`
		Index       int         `json:"index"`
		Hram        string      `json:"hram"`
		Random      string      `json:"random"`
	}
	gin_helper.BindJson(c, &input)

	var randKey crypto.Key
	{
		random, err := imp.store.ReadProperty(c, fmt.Sprintf("random_%s", input.Random))
		if err != nil {
			gin_helper.FailError(c, err)
			return
		}
		bts, _ := hex.DecodeString(random)
		copy(randKey[:], bts[:])
	}

	t, err := mint.ReadTransaction(input.Transaction.String())
	if err != nil {
		gin_helper.FailError(c, err)
		return
	}

	if input.Index >= len(t.Outputs) {
		gin_helper.FailError(c, errors.New("index exceeds output bounds"))
		return
	}

	mask := t.Outputs[input.Index].Mask
	var hram [32]byte
	{
		bts, err := hex.DecodeString(input.Hram)
		if err != nil {
			gin_helper.FailError(c, errors.New("index exceeds output bounds"))
			return
		}
		copy(hram[:], bts[:])
	}
	resp := imp.key.Response(hram, &mask, &randKey)
	gin_helper.OK(c, "response", hex.EncodeToString(resp[:]))
}
