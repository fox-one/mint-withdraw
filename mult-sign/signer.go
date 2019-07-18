package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mint-withdraw"
	"github.com/fox-one/mint-withdraw/store"
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
	extra    string
}

func newSigner(cachePath, sigKey, receiver, receiverExtra string, outputIndex int, signerAPIBases ...string) (*signer, error) {
	s, err := store.NewStore(cachePath)
	if err != nil {
		return nil, err
	}

	k, err := NewKey(outputIndex, sigKey, signerAPIBases...)
	if err != nil {
		return nil, err
	}

	return &signer{
		key:      k,
		store:    s,
		receiver: receiver,
		extra:    receiverExtra,
	}, nil
}

func (s signer) pledgeTransaction(ctx context.Context, assetID, signerSpendPub, payeeSpendPub string, transactions []string) error {
	asset, err := crypto.HashFromString(assetID)
	if err != nil {
		return err
	}

	t := common.NewTransaction(asset)

	{
		extra, err := hex.DecodeString(signerSpendPub + payeeSpendPub)
		if err != nil {
			return err
		}
		t.Extra = extra
	}

	for _, s := range transactions {
		h, err := crypto.HashFromString(s)
		if err != nil {
			return err
		}
		t.AddInput(h, 0)
	}

	seed := make([]byte, 64)
	_, err = rand.Read(seed)
	if err != nil {
		return err
	}

	t.AddOutputWithType(common.OutputTypeNodePledge, nil, common.Script{}, common.NewInteger(10000), seed)

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

	if _, err := mint.WithdrawTransaction(ctx, t, s.key, s.store, s.receiver, s.extra); err != nil {
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
