package main

import (
	"context"
	"os"
	"time"

	"github.com/fox-one/mint-withdraw"
	"github.com/fox-one/mint-withdraw/store"
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
	extra    string
}

func newSigner() (*signer, error) {
	s, err := store.NewStore(cachePath)
	if err != nil {
		return nil, err
	}

	k, err := NewKey(View, Spend)
	if err != nil {
		return nil, err
	}

	return &signer{
		key:      k,
		store:    s,
		receiver: Receiver,
		extra:    ReceiverExtra,
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
