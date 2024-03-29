package main

import (
	"crypto/rand"
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

func parseKey(s string) (crypto.Key, error) {
	var k crypto.Key
	b, err := hex.DecodeString(s)
	if err != nil {
		return k, err
	}

	copy(k[:32], b[:32])
	return k, nil
}

// NewKey create key
func NewKey(view, spend string) (*Key, error) {
	key := &Key{}
	{
		k, err := parseKey(spend)
		if err != nil {
			return nil, err
		}
		key.Spend = k
	}

	if view != "" {
		k, err := parseKey(view)
		if err != nil {
			return nil, err
		}
		key.View = k
	} else {
		key.View = key.Spend.Public().DeterministicHashDerive()
	}
	return key, nil
}

// Accounts accounts
func (k Key) Accounts() []*common.Address {
	addr := &common.Address{
		PrivateSpendKey: k.Spend,
		PrivateViewKey:  k.View,
		PublicSpendKey:  k.Spend.Public(),
		PublicViewKey:   k.View.Public(),
	}
	return []*common.Address{addr}
}

// VerifyOutputs verify ouputs
func (k Key) VerifyOutputs(t *mint.Transaction) ([]int, error) {
	var outputs = make([]int, 0, len(t.Outputs))
	for idx, o := range t.Outputs {
		for _, key := range o.Keys {
			// if mixin.ViewGhostOutputKey((*mixin.Key)(key), (*mixin.Key)(&k.View), (*mixin.Key)(&o.Mask), idx).String() == k.Spend.Public().String() {
			if crypto.ViewGhostOutputKey(key, &k.View, &o.Mask, uint64(idx)).String() == k.Spend.Public().String() {
				outputs = append(outputs, idx)
				break
			}
		}
	}
	return outputs, nil
}

// Sign sign transaction, only for transaction, not for mint/deposit
func (k Key) Sign(out *common.Transaction, t *mint.Transaction) (*common.VersionedTransaction, error) {
	return k.AggregateSign(out, t)
}

func (k Key) SignMap(out *common.Transaction, t *mint.Transaction) (*common.VersionedTransaction, error) {
	signed := out.AsVersioned()

	for i := range signed.Inputs {
		err := signed.SignInput(t, i, k.Accounts())
		if err != nil {
			return nil, err
		}
	}
	return signed, nil
}

func (k Key) AggregateSign(out *common.Transaction, t *mint.Transaction) (*common.VersionedTransaction, error) {
	signed := out.AsVersioned()

	var seed = make([]byte, 32)
	rand.Reader.Read(seed)
	if err := signed.AggregateSign(t, [][]*common.Address{k.Accounts()}, seed); err != nil {
		return nil, err
	}
	return signed, nil
}
