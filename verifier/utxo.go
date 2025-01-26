package main

import (
	"context"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
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
	err := mixinnet.NewClient(mixinnet.DefaultSafeConfig).CallMixinNetRPC(ctx, &utxo, "getutxo", hash, index)
	return &utxo, err
}
