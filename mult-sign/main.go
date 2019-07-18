package main

import (
	"context"
	"crypto/rand"
	"os"
	"time"

	"github.com/MixinNetwork/mixin/crypto"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

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
			cli.StringSliceFlag{Name: "spends, s"},
		},
		Action: encodeAddress,
	})

	app.Commands = append(app.Commands, cli.Command{
		Name: "pledge",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "asset, a"},
			cli.StringSliceFlag{Name: "transaction, t"},
			cli.IntFlag{Name: "index, i"},
			cli.StringFlag{Name: "signer-spend-pub, ss"},
			cli.StringFlag{Name: "payee-spend-pub, ps"},
		},
		Action: func(c *cli.Context) error {
			s, err := newSigner(cachePath, spendPub, view, sigKey, receiver, receiverExtra, signerAPIBases...)
			if err != nil {
				return err
			}
			return s.pledgeTransaction(ctx,
				c.String("asset"),
				c.String("signer-spend-pub"), c.String("payee-spend-pub"),
				c.StringSlice("transaction"))
		},
	})

	app.Commands = append(app.Commands, cli.Command{
		Name: "transaction",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "transaction, t"},
		},
		Action: func(c *cli.Context) error {
			s, err := newSigner(cachePath, spendPub, view, sigKey, receiver, receiverExtra, signerAPIBases...)
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
		},
		Action: func(c *cli.Context) error {
			s, err := newSigner(cachePath, spendPub, view, sigKey, receiver, receiverExtra, signerAPIBases...)
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
