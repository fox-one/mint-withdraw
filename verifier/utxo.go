package main

import (
	"context"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mixin-sdk-go"
)

type (
	UTXOWithLock struct {
		common.UTXO
		LockHash crypto.Hash `json:"lock"`
	}
)

func ReadUTXOLock(hash crypto.Hash, index uint) (*UTXOWithLock, error) {
	var utxo UTXOWithLock
	ctx := context.Background()
	ctx = mixin.WithMixinNetHost(ctx, "http://mixin-node-box-1.b.watch:8239")
	err := mixin.CallMixinNetRPC(ctx, &utxo, "getutxo", hash, index)
	return &utxo, err
}
