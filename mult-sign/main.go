package main

import (
	"context"
	"crypto/rand"
	"os"
	"time"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/fox-one/mint-withdraw"
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
	store    *Store
	receiver string
	extra    string
}

func newSigner(outputIndex int) (*signer, error) {
	s, err := newStore(cachePath)
	if err != nil {
		return nil, err
	}

	return &signer{
		key:      NewKey(outputIndex, signerAPIBases...),
		store:    s,
		receiver: receiver,
		extra:    receiverExtra,
	}, nil
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

	ensureFunc(func() error {
		err := s.withdrawTransaction(ctx, ds[0].Transaction.String())
		if err != nil {
			log.Errorln("withdraw transaction", err)
			return err
		}

		ensureFunc(func() error {
			err := s.store.writeBatch(batch + 1)
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
	app.Name = "mult-sign"
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
		Name: "random-key",
		Action: func(c *cli.Context) error {
			randomFunc := func() crypto.Key {
				seed := make([]byte, 64)
				rand.Read(seed)
				return crypto.NewKeyFromSeed(seed)
			}

			k := randomFunc()
			log.Println("private", k)
			log.Println("public", k.Public())
			return nil
		},
	})

	app.Commands = append(app.Commands, cli.Command{
		Name: "transaction",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "transaction, t"},
			cli.IntFlag{Name: "index, i"},
		},
		Action: func(c *cli.Context) error {
			s, err := newSigner(c.Int("index"))
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
			s, err := newSigner(c.Int("index"))
			if err != nil {
				return err
			}
			if v := c.Uint64("from"); v > 0 {
				s.store.writeBatch(v)
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
