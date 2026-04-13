package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/MixinNetwork/mixin/rpc"
	"github.com/fox-one/mint-withdraw"
	"github.com/fox-one/mint-withdraw/store"
	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/fox-one/pkg/uuid"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

type signer struct {
	key      *Key
	store    *store.Store
	walletID string

	user *mixin.Client
}

func (s signer) readTransaction(ctx context.Context, transaction string) (*mint.Transaction, error) {
	storeKey := fmt.Sprintf("gettransaction_%s", transaction)
	if s.store != nil {
		if cached, err := s.store.ReadProperty(ctx, storeKey); err == nil && strings.TrimSpace(cached) != "" {
			var t mint.Transaction
			if err := jsoniter.Unmarshal([]byte(cached), &t); err == nil && t.Hash.HasValue() {
				return &t, nil
			}
		}
	}

	t, err := mint.ReadTransaction(transaction)
	if err != nil {
		return nil, err
	}
	if s.store != nil {
		if bts, err := jsoniter.Marshal(t); err == nil {
			if err := s.store.WriteProperty(ctx, storeKey, string(bts)); err != nil {
				log.WithError(err).Warnln("cache gettransaction", transaction)
			}
		}
	}
	return t, nil
}

func (s signer) withdrawTransaction(ctx context.Context, txHash string) error {
	t, err := s.readTransaction(ctx, txHash)
	if err != nil {
		fmt.Println("read transaction", err)
		return err
	}

	mask, keys, err := s.createGhostOutput(ctx)
	if err != nil {
		return err
	}

	if _, err := mint.WithdrawTransaction(ctx, t, s.key, s.store, mask, keys, s.walletID); err != nil {
		fmt.Println("withdraw transaction", err)
		return err
	}

	return nil
}

func (s signer) withdrawMintTransaction(ctx context.Context, txHash string) error {
	t, err := s.readTransaction(ctx, txHash)
	if err != nil {
		return err
	}

	mask, keys, err := s.createGhostOutput(ctx)
	if err != nil {
		return err
	}

	if _, err := mint.WithdrawTransaction(ctx, t, s.key, s.store, mask, keys, s.walletID); err != nil {
		return err
	}

	lockHash, err := mint.ReadUTXOLockWithCache(ctx, s.store, t.Hash, len(t.Outputs)-2)
	if err != nil {
		return err
	}

	if !lockHash.HasValue() {
		return fmt.Errorf("safe utxo (%v) not dispatched", lockHash)
	}

	return s.withdrawTransaction(ctx, lockHash.String())
}

func (s signer) createGhostOutput(ctx context.Context) (crypto.Key, []*crypto.Key, error) {
	if s.user == nil || s.walletID == "" {
		return crypto.Key{}, nil, errors.New("no valid output wallet")
	}

	ghosts, err := s.user.SafeCreateGhostKeys(ctx, []*mixin.GhostInput{
		{
			Receivers: []string{s.walletID},
			Index:     0,
			Hint:      uuid.New(),
		},
	})
	if err != nil {
		fmt.Println("safe create ghost keys", err)
		return crypto.Key{}, nil, err
	}

	mask, err := parseKey(ghosts[0].Mask.String())
	if err != nil {
		fmt.Println("parse mask", err)
		return crypto.Key{}, nil, err
	}
	key, err := parseKey(ghosts[0].Keys[0].String())
	if err != nil {
		fmt.Println("parse key", err)
		return crypto.Key{}, nil, err
	}
	return mask, []*crypto.Key{&key}, nil
}

func (s signer) mintWithdraw(ctx context.Context) error {
	batch := s.store.Batch()

	ds, err := mint.ListMintDistributions(batch, 1)
	if err != nil {
		return err
	}

	log.Infoln("mint batch", batch, len(ds))
	if len(ds) == 0 {
		return nil
	}

	log.Debugln("withdraw transaction", ds[0].Transaction)
	ensureFunc(func() error {
		err := s.withdrawMintTransaction(ctx, ds[0].Transaction.String())
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

func (s signer) pledgeTransaction(ctx context.Context, keystore, signerSpendPub, payeeSpendPub, txHash string, dryRun bool) error {
	in, err := s.readTransaction(ctx, txHash)
	if err != nil {
		return err
	}

	if keystore != "" {
		bts, err := os.ReadFile(keystore)
		if err != nil {
			return err
		}
		var keys struct {
			Signer *crypto.Key `json:"s"`
			Payee  *crypto.Key `json:"p"`
		}
		if err := json.Unmarshal(bts, &keys); err != nil {
			return err
		}
		if keys.Signer == nil || !keys.Signer.HasValue() ||
			keys.Payee == nil || !keys.Payee.HasValue() {
			return errors.New("unmarshal keystore failed")
		}
		signerSpendPub = keys.Signer.Public().String()
		payeeSpendPub = keys.Payee.Public().String()
	}

	t := common.NewTransactionV5(in.Asset)
	{
		extra, err := hex.DecodeString(signerSpendPub + payeeSpendPub)
		if err != nil {
			return err
		}
		t.Extra = extra
	}

	amount := common.NewInteger(0)
	utxos, err := s.key.VerifyOutputs(ctx, s.store, in)
	if err != nil {
		return err
	}
	for _, i := range utxos {
		t.AddInput(in.Hash, uint(i))
		amount = amount.Add(in.Outputs[i].Amount)
	}

	seed := make([]byte, 64)
	_, err = rand.Read(seed)
	if err != nil {
		return err
	}

	t.AddOutputWithType(common.OutputTypeNodePledge, nil, common.Script{}, amount, seed)
	{
		node := mint.RandomNode()
		info, err := rpc.GetInfo(node)
		if err != nil {
			return err
		}
		snap, err := rpc.GetSnapshot(node, info.Consensus.String())
		if err != nil {
			return err
		}
		t.References = []crypto.Hash{snap.SoleTransaction()}
	}

	log.Println("begin to sign")
	signed, err := s.key.Sign(t, in)
	if err != nil {
		return err
	}

	log.Println("signed")
	rawData := hex.EncodeToString(signed.Marshal())

	if dryRun {
		bts, _ := jsoniter.MarshalIndent(signed, "", "    ")
		log.Println(string(bts))
		log.Println(rawData)
		return nil
	}

	out, err := mint.DoTransaction(ctx, rawData)
	if out != nil {
		log.Println(out.Hash)
	}
	return err
}
