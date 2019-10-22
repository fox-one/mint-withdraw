package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mint-withdraw"
	"github.com/fox-one/mint-withdraw/store"
	"github.com/fox-one/mixin-sdk/mixin"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
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
			cli.IntFlag{Name: "index, i"},
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
					time.Sleep(time.Minute * 5)
					continue
				}
				log.Errorln("mint withdraw", err)
				time.Sleep(time.Minute)
			}

			return nil
		},
	})

	if err := app.Run(os.Args); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
