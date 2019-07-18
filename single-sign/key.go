package main

import (
	"encoding/hex"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mint-withdraw"
)

// Key key
type Key struct {
	Spend crypto.Key
	View  crypto.Key
}

// NewKey create key
func NewKey(view, spend string) (*Key, error) {
	decodeFunc := func(s string) (crypto.Key, error) {
		var k crypto.Key
		b, err := hex.DecodeString(s)
		if err != nil {
			return k, err
		}

		copy(k[:32], b[:32])
		return k, nil
	}
	key := &Key{}
	{
		k, err := decodeFunc(view)
		if err != nil {
			return nil, err
		}
		key.View = k
	}

	{
		k, err := decodeFunc(spend)
		if err != nil {
			return nil, err
		}
		key.Spend = k
	}
	return key, nil
}

// Accounts accounts
func (k Key) Accounts() []common.Address {
	addr := common.Address{
		PrivateSpendKey: k.Spend,
		PrivateViewKey:  k.View,
		PublicSpendKey:  k.Spend.Public(),
		PublicViewKey:   k.View.Public(),
	}
	return []common.Address{addr}
}

// VerifyOutputs verify ouputs
func (k Key) VerifyOutputs(t *mint.Transaction) ([]int, error) {
	var outputs = make([]int, 0, len(t.Outputs))
	for idx, o := range t.Outputs {
		for _, key := range o.Keys {
			if crypto.ViewGhostOutputKey(&key, &k.View, &o.Mask, uint64(idx)).String() == k.Spend.Public().String() {
				outputs = append(outputs, idx)
				break
			}
		}
	}
	return outputs, nil
}

// Sign sign transaction, only for transaction, not for mint/deposit
func (k Key) Sign(out *common.Transaction, t *mint.Transaction) (*common.VersionedTransaction, error) {
	signed := out.AsLatestVersion()

	for i := range signed.Inputs {
		err := signed.SignInput(t, i, k.Accounts())
		if err != nil {
			return nil, err
		}
	}
	return signed, nil
}
