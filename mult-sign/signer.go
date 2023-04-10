package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mint-withdraw"
	"github.com/fox-one/mint-withdraw/store"
	"github.com/fox-one/mixin-sdk"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

func ensureFunc(f func() error) {
	for {
		if err := f(); err == nil {
			return
		}
		time.Sleep(time.Second)
	}
}

type signer struct {
	key      *Key
	store    *store.Store
	receiver string
	walletID string

	user *mixin.User
}

func newSigner(cachePath, spendPub, view, sigKey, receiver, clientID, sessionID, sessionKey, receiverExtra string, signerAPIBases ...string) (*signer, error) {
	s, err := store.NewStore(cachePath)
	if err != nil {
		return nil, err
	}

	k, err := NewKey(spendPub, view, sigKey, signerAPIBases...)
	if err != nil {
		return nil, err
	}

	signer := signer{
		key:      k,
		store:    s,
		receiver: receiver,
		walletID: receiverExtra,
	}

	if clientID != "" && sessionID != "" && sessionKey != "" {
		u, err := mixin.NewUser(clientID, sessionID, sessionKey)
		if err != nil {
			return nil, err
		}
		signer.user = u
	}

	if signer.receiver == "" && (signer.user == nil && signer.walletID == "") {
		return nil, errors.New("no valid output account")
	}

	return &signer, nil
}

func (s signer) pledgeTransaction(ctx context.Context, assetID, signerSpendPub, payeeSpendPub string, transactions []string) error {
	asset, err := crypto.HashFromString(assetID)
	if err != nil {
		return err
	}

	t := common.NewTransactionV3(asset)

	{
		extra, err := hex.DecodeString(signerSpendPub + payeeSpendPub)
		if err != nil {
			return err
		}
		t.Extra = extra
	}

	amount := common.NewInteger(0)
	for _, h := range transactions {
		in, err := mint.ReadTransaction(h)
		if err != nil {
			return err
		}
		os, err := s.key.VerifyOutputs(in)
		if err != nil {
			return err
		}
		for _, i := range os {
			t.AddInput(in.Hash, i)
			amount = amount.Add(in.Outputs[i].Amount)
		}
	}

	seed := make([]byte, 64)
	_, err = rand.Read(seed)
	if err != nil {
		return err
	}

	t.AddOutputWithType(common.OutputTypeNodePledge, nil, common.Script{}, amount, seed)

	signed, err := s.key.Sign(t, nil)
	if err != nil {
		return err
	}

	rawData := hex.EncodeToString(signed.Marshal())
	{
		bts, _ := jsoniter.Marshal(signed)
		log.Println(string(bts))
		log.Println(rawData)
	}

	out, err := mint.DoTransaction(ctx, rawData)
	if out != nil {
		log.Println(out.Hash)
	}
	return err
}

func (s signer) withdrawTransaction(ctx context.Context, transaction string) error {
	t, err := mint.ReadTransaction(transaction)
	if err != nil {
		return err
	}

	var (
		mask crypto.Key
		keys []crypto.Key
	)
	if s.receiver == "" {
		output, err := s.user.MakeTransactionOutput(ctx, s.walletID)
		if err != nil {
			return err
		}
		m, err := decodeKey(output.Mask)
		if err != nil {
			return err
		}
		key, err := decodeKey(output.Keys[0])
		if err != nil {
			return err
		}
		mask = *m
		keys = []crypto.Key{*key}
	}

	if _, err := mint.WithdrawTransaction(ctx, t, s.key, s.store, s.receiver, mask, keys, s.walletID); err != nil {
		return err
	}

	return nil
}

func (s signer) mintWithdraw(ctx context.Context) error {
	batch := s.store.Batch()

	ds, err := mint.ListMintDistributions(batch, 1)
	if err != nil {
		return err
	}

	if len(ds) == 0 {
		return nil
	}

	log.Debugln("withdraw transaction", ds[0].Transaction)
	ensureFunc(func() error {
		err := s.withdrawTransaction(ctx, ds[0].Transaction.String())
		if err != nil {
			log.Errorln("withdraw transaction", err)
			return err
		}

		ensureFunc(func() error {
			err := s.store.WriteBatch(ds[0].Batch + 1)
			if err != nil {
				log.Errorln("write batch", err)
			}
			return err
		})
		return nil
	})

	return nil
}
