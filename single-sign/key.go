package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mint-withdraw"
)

// Key key
type Key struct {
	accounts []*common.Address
}

func parseKey(s string) (crypto.Key, error) {
	var k crypto.Key
	b, err := hex.DecodeString(s)
	if err != nil {
		return k, err
	}
	if len(b) != 32 {
		return k, fmt.Errorf("invalid key length: got %d bytes", len(b))
	}

	copy(k[:], b)
	return k, nil
}

type matchedAccount struct {
	keyIndex int
	account  *common.Address
}

// NewKey create key
func NewKey(spends []string) (*Key, error) {
	key := &Key{}
	for _, spend := range spends {
		spendKey, err := parseKey(spend)
		if err != nil {
			return nil, err
		}
		viewKey := spendKey.Public().DeterministicHashDerive()
		key.accounts = append(key.accounts, &common.Address{
			PrivateSpendKey: spendKey,
			PrivateViewKey:  viewKey,
			PublicSpendKey:  spendKey.Public(),
			PublicViewKey:   viewKey.Public(),
		})
	}
	if len(key.accounts) == 0 {
		return nil, fmt.Errorf("missing spend keys")
	}
	return key, nil
}

// Accounts accounts
func (k Key) Accounts() []*common.Address {
	return append([]*common.Address{}, k.accounts...)
}

func (k Key) CacheID() string {
	accounts := k.Accounts()
	if len(accounts) == 1 {
		return accounts[0].String()
	}

	h := sha256.New()
	for _, account := range accounts {
		_, _ = h.Write([]byte(account.PublicSpendKey.String()))
		_, _ = h.Write([]byte{0})
	}
	return "multi_" + hex.EncodeToString(h.Sum(nil)[:16])
}

// VerifyOutputs verify ouputs
func (k Key) VerifyOutputs(ctx context.Context, store mint.Store, t *mint.Transaction) ([]int, error) {
	var outputs = make([]int, 0, len(t.Outputs))
	for idx, o := range t.Outputs {
		if len(k.matchingAccounts(o.Keys, o.Mask, uint64(idx))) == 0 {
			continue
		}

		lockHash, err := mint.ReadUTXOLockWithCache(ctx, store, t.Hash, idx)
		if err != nil {
			return nil, err
		}
		if lockHash.HasValue() {
			continue
		}
		outputs = append(outputs, idx)
	}
	return outputs, nil
}

// Sign sign transaction, only for transaction, not for mint/deposit
func (k Key) Sign(out *common.Transaction, t *mint.Transaction) (*common.VersionedTransaction, error) {
	return k.AggregateSign(out, t)
}

func (k Key) SignMap(out *common.Transaction, t *mint.Transaction) (*common.VersionedTransaction, error) {
	signed := out.AsVersioned()
	accounts, err := k.accountsByInput(signed, t)
	if err != nil {
		return nil, err
	}

	for i := range signed.Inputs {
		if err := signed.SignInput(t, i, accounts[i]); err != nil {
			return nil, err
		}
	}
	return signed, nil
}

func (k Key) AggregateSign(out *common.Transaction, t *mint.Transaction) (*common.VersionedTransaction, error) {
	signed := out.AsVersioned()
	accounts, err := k.accountsByInput(signed, t)
	if err != nil {
		return nil, err
	}

	var seed = make([]byte, 32)
	rand.Reader.Read(seed)
	if err := signed.AggregateSign(t, accounts, seed); err != nil {
		return nil, err
	}
	return signed, nil
}

func (k Key) accountsByInput(signed *common.VersionedTransaction, t *mint.Transaction) ([][]*common.Address, error) {
	accounts := make([][]*common.Address, len(signed.Inputs))
	for idx, input := range signed.Inputs {
		utxo, err := t.ReadUTXOKeys(input.Hash, input.Index)
		if err != nil {
			return nil, err
		}
		accounts[idx] = k.matchingAccounts(utxo.Keys, utxo.Mask, uint64(input.Index))
		if len(accounts[idx]) == 0 {
			return nil, fmt.Errorf("no signer for input %s:%d", input.Hash, input.Index)
		}
	}
	return accounts, nil
}

func (k Key) matchingAccounts(keys []*crypto.Key, mask crypto.Key, index uint64) []*common.Address {
	keysFilter := make(map[string]int, len(keys))
	for idx, key := range keys {
		keysFilter[key.String()] = idx
	}

	var matches []matchedAccount
	for _, account := range k.accounts {
		priv := crypto.DeriveGhostPrivateKey(&mask, &account.PrivateViewKey, &account.PrivateSpendKey, index)
		if keyIndex, found := keysFilter[priv.Public().String()]; found {
			matches = append(matches, matchedAccount{
				keyIndex: keyIndex,
				account:  account,
			})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].keyIndex < matches[j].keyIndex
	})

	accounts := make([]*common.Address, 0, len(matches))
	for _, match := range matches {
		accounts = append(accounts, match.account)
	}
	return accounts
}
