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

	"github.com/MixinNetwork/mixin/common"
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
		Transaction string        `json:"transaction"`
		Index       int           `json:"index"`
		Randoms     []*crypto.Key `json:"randoms"`
	}
	gin_helper.BindJson(c, &input)

	message, err := hex.DecodeString(input.Transaction)
	if err != nil {
		gin_helper.FailError(c, err)
		return
	}

	var transaction common.Transaction
	{
		t, err := common.UnmarshalVersionedTransaction(message)
		if err != nil {
			gin_helper.FailError(c, err)
			return
		}

		transaction = t.Transaction
	}

	outputAmount := common.NewInteger(0)
	for _, output := range transaction.Outputs {
		if _, found := acceptedOutputTypes[output.Type]; !found {
			gin_helper.FailError(c, errors.New("output not accepted"))
			return
		}

		outputAmount = outputAmount.Add(output.Amount)
	}
	if outputAmount.Cmp(maxOutputAmount) > 0 {
		gin_helper.FailError(c, errors.New("output amount too large"))
		return
	}

	var randKey *crypto.Key
	for _, r := range input.Randoms {
		random, err := imp.store.ReadProperty(c, fmt.Sprintf("random_%s", r.String()))
		if err != nil {
			if err == ErrNotFound {
				continue
			}

			gin_helper.FailError(c, err)
			return
		}

		if random != "" {
			bts, _ := hex.DecodeString(random)
			randKey = &crypto.Key{}
			copy(randKey[:], bts[:])
			break
		}
	}

	if randKey == nil {
		gin_helper.FailError(c, errors.New("invalid random"))
		return
	}

	if input.Index >= len(transaction.Inputs) {
		gin_helper.FailError(c, errors.New("index exceeds input bounds"))
		return
	}

	inputTran := transaction.Inputs[input.Index]
	t, err := mint.ReadTransaction(inputTran.Hash.String())
	if err != nil {
		gin_helper.FailError(c, err)
		return
	}

	if inputTran.Index >= len(t.Outputs) {
		gin_helper.FailError(c, errors.New("index exceeds output bounds"))
		return
	}

	utxo := t.Outputs[inputTran.Index]
	hram := challenge(&utxo.Keys[0], message, input.Randoms...)
	resp := imp.key.Response(hram, &utxo.Mask, randKey, uint64(inputTran.Index))
	gin_helper.OK(c, "response", hex.EncodeToString(resp[:]))
}
