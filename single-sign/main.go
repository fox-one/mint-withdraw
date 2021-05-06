package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mint-withdraw"
	"github.com/fox-one/mint-withdraw/store"
	mixin "github.com/fox-one/mixin-sdk"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var commands []cli.Command

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

func newSigner() (*signer, error) {
	signer := signer{
		receiver: Address,
		walletID: ReceiverWallet,
	}
	s, err := store.NewStore(cachePath)
	if err != nil {
		return nil, err
	}
	signer.store = s

	k, err := NewKey(View, Spend)
	if err != nil {
		return nil, err
	}
	signer.key = k

	if ClientID != "" && SessionID != "" && SessionKey != "" {
		u, err := mixin.NewUser(ClientID, SessionID, SessionKey)
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

func (s signer) withdrawTransaction(ctx context.Context, transaction string) error {
	t, err := mint.ReadTransaction(transaction)
	if err != nil {
		return err
	}

	receiver := s.receiver
	extra := s.walletID
	var mask crypto.Key
	var keys []crypto.Key

	if receiver == "" {
		output, err := s.user.MakeTransactionOutput(ctx, s.walletID)
		if err != nil {
			return err
		}
		m, err := parseKey(output.Mask)
		if err != nil {
			return err
		}
		key, err := parseKey(output.Keys[0])
		if err != nil {
			return err
		}
		mask = m
		keys = []crypto.Key{key}
	}

	if _, err := mint.WithdrawTransaction(ctx, t, s.key, s.store, receiver, mask, keys, extra); err != nil {
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

func (s signer) pledgeTransaction(ctx context.Context, keystore, signerSpendPub, payeeSpendPub, transaction string, dryRun bool) error {
	in, err := mint.ReadTransaction(transaction)
	if err != nil {
		return err
	}

	if keystore != "" {
		bts, err := ioutil.ReadFile(keystore)
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
		if keys.Signer == nil || keys.Payee == nil {
			return errors.New("unmarshal keystore failed")
		}
		signerSpendPub = keys.Signer.Public().String()
		payeeSpendPub = keys.Payee.Public().String()
	}

	t := common.NewTransaction(in.Asset)
	{
		extra, err := hex.DecodeString(signerSpendPub + payeeSpendPub)
		if err != nil {
			return err
		}
		t.Extra = extra
	}

	amount := common.NewInteger(0)
	os, err := s.key.VerifyOutputs(in)
	if err != nil {
		return err
	}
	for _, i := range os {
		t.AddInput(in.Hash, i)
		amount = amount.Add(in.Outputs[i].Amount)
	}

	seed := make([]byte, 64)
	_, err = rand.Read(seed)
	if err != nil {
		return err
	}

	t.AddOutputWithType(common.OutputTypeNodePledge, nil, common.Script{}, amount, seed)

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

func main() {
	ctx := context.Background()

	app := cli.NewApp()
	app.Name = "single-sign"
	app.Version = "1.0.0"

	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "debug"},
	}

	app.Before = func(c *cli.Context) error {
		if c.GlobalBool("debug") {
			log.SetLevel(log.DebugLevel)
		}

		return nil
	}

	app.Commands = append(app.Commands, cli.Command{
		Name: "print",
		Action: func(c *cli.Context) error {
			s, err := newSigner()
			if err != nil {
				return err
			}

			fmt.Printf("Address: %s\nPrivate View: %s\nPublic Spend: %s",
				s.key.Accounts()[0], s.key.View, s.key.Spend.Public())

			return nil
		},
	})

	app.Commands = append(app.Commands, cli.Command{
		Name: "transaction",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "transaction, t"},
		},
		Action: func(c *cli.Context) error {
			s, err := newSigner()
			if err != nil {
				return err
			}
			return s.withdrawTransaction(ctx, c.String("transaction"))
		},
	})

	app.Commands = append(app.Commands, cli.Command{
		Name: "mint",
		Flags: []cli.Flag{
			cli.Uint64Flag{Name: "from, f"},
			cli.Int64Flag{Name: "duration"},
		},
		Action: func(c *cli.Context) error {
			s, err := newSigner()
			if err != nil {
				return err
			}
			if v := c.Uint64("from"); v > 0 {
				s.store.WriteBatch(v)
			}

			for {
				err := s.mintWithdraw(ctx)
				if err == nil {
					dur := time.Minute
					if v := c.Int64("duration"); v > 0 {
						dur = time.Duration(v) * time.Second
					}
					time.Sleep(dur)
					continue
				}
				log.Errorln("mint withdraw", err)
				time.Sleep(time.Minute)
			}

			return nil
		},
	})

	app.Commands = append(app.Commands, cli.Command{
		Name: "pledge",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "transaction, t", Required: true},
			cli.BoolFlag{Name: "dry"},
			cli.StringFlag{Name: "asset, a"},
			cli.StringFlag{Name: "keystore, k"},
			cli.StringFlag{Name: "signer-spend-pub, ss"},
			cli.StringFlag{Name: "payee-spend-pub, ps"},
		},
		Action: func(c *cli.Context) error {
			s, err := newSigner()
			if err != nil {
				return err
			}
			return s.pledgeTransaction(ctx,
				c.String("keystore"),
				c.String("signer-spend-pub"),
				c.String("payee-spend-pub"),
				c.String("transaction"),
				c.Bool("dry"))
		},
	})

	app.Commands = append(app.Commands, commands...)
	if err := app.Run(os.Args); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
