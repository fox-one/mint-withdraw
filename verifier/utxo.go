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
	err := mixin.CallMixinNetRPC(context.Background(), &utxo, "getutxo", hash, index)
	return &utxo, err
}
