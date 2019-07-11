package main

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/gin-contrib/gin_helper"
	"github.com/fox-one/mint-withdraw"
	"github.com/gin-gonic/gin"
)

type serverImp struct {
	store *Store
	key   *Key
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

func main() {
	k, err := NewKey(View, Spend, CoSignerCount)
	if err != nil {
		panic(err)
	}

	imp := serverImp{
		store: NewStore(),
		key:   k,
	}

	r := gin.Default()
	r.GET("/hc", func(c *gin.Context) {
		gin_helper.OK(c)
	})

	r.POST("/random", imp.random)

	r.POST("/sign", imp.sign)

	r.Run(fmt.Sprintf(":%d", Port))
}
