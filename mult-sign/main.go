package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"time"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/btcsuite/btcutil/base58"
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

func newSigner(outputIndex int) (*signer, error) {
	s, err := store.NewStore(cachePath)
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
		Name: "address",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "view, v"},
			cli.StringSliceFlag{Name: "spends, s"},
		},
		Action: func(c *cli.Context) error {
			addressFunc := func(spendPub, viewPub crypto.Key) string {
				const MainNetworkID = "XIN"
				data := append([]byte(MainNetworkID), spendPub[:]...)
				data = append(data, viewPub[:]...)
				checksum := crypto.NewHash(data)
				data = append(spendPub[:], viewPub[:]...)
				data = append(data, checksum[:4]...)
				return MainNetworkID + base58.Encode(data)
			}

			decodeKey := func(s string) (*crypto.Key, error) {
				log.Println(s)
				var k crypto.Key

				b, err := hex.DecodeString(s)
				if err != nil {
					return nil, err
				}
				copy(k[:], b[:])
				return &k, nil
			}
			viewPub, err := decodeKey(c.String("view"))
			if err != nil {
				return err
			}

			var spendPub *crypto.Key
			for idx, s := range c.StringSlice("spends") {
				p, err := decodeKey(s)
				if err != nil {
					return err
				}

				if idx == 0 {
					spendPub = p
				} else {
					spendPub = crypto.KeyAddPub(spendPub, p)
				}
			}
			log.Println("address", addressFunc(*spendPub, *viewPub))
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
