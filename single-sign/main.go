package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

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

func configReader(configFile string, stdin bool) (io.Reader, error) {
	if !stdin && strings.TrimSpace(configFile) != "-" {
		return nil, nil
	}

	if !isStdinAvailable() {
		return nil, fmt.Errorf("stdin config requested but stdin is not available")
	}
	return os.Stdin, nil
}

func isStdinAvailable() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice == 0
}

func main() {
	ctx := context.Background()

	app := cli.NewApp()
	app.Name = "single-sign"
	app.Version = "1.0.0"
	configFile := ""
	configStdin := false
	spendIndex := uint64(0)

	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "debug"},
		cli.StringFlag{Name: "config"},
		cli.BoolFlag{Name: "stdin"},
		cli.Uint64Flag{Name: "spend-index"},
	}

	app.Before = func(c *cli.Context) error {
		if c.GlobalBool("debug") {
			log.SetLevel(log.DebugLevel)
		}

		if f := c.GlobalString("config"); f != "" {
			configFile = strings.TrimSpace(f)
		}
		configStdin = c.GlobalBool("stdin")
		spendIndex = c.GlobalUint64("spend-index")

		return nil
	}

	newCommandSigner := func() (*signer, error) {
		r, err := configReader(configFile, configStdin)
		if err != nil {
			return nil, err
		}
		return newSigner(configFile, r, spendIndex)
	}

	app.Commands = append(app.Commands, cli.Command{
		Name: "print",
		Action: func(c *cli.Context) error {
			s, err := newCommandSigner()
			if err != nil {
				return err
			}

			for idx, account := range s.key.Accounts() {
				fmt.Printf("Address[%d]: %s\nPrivate View[%d]: %s\nPublic Spend[%d]: %s\n",
					idx+1, account, idx+1, account.PrivateViewKey, idx+1, account.PublicSpendKey)
			}

			return nil
		},
	})

	app.Commands = append(app.Commands, cli.Command{
		Name: "transaction",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "transaction, t"},
		},
		Action: func(c *cli.Context) error {
			s, err := newCommandSigner()
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
			s, err := newCommandSigner()
			if err != nil {
				return err
			}
			if v := c.Uint64("from"); v > 0 {
				s.store.WriteBatch(v)
			}

			for {
				err := s.mintWithdraw(ctx)
				if err == nil {
					dur := time.Second
					if v := c.Int64("duration"); v > 0 {
						dur = time.Duration(v) * time.Second
					}
					time.Sleep(dur)
					continue
				}
				log.Errorln("mint withdraw", err)
				time.Sleep(time.Second)
			}
		},
	})

	app.Commands = append(app.Commands, cli.Command{
		Name: "pledge",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "transaction, t", Required: true},
			cli.BoolFlag{Name: "dry"},
			cli.StringFlag{Name: "keystore, k"},
			cli.StringFlag{Name: "signer-spend-pub, ss"},
			cli.StringFlag{Name: "payee-spend-pub, ps"},
		},
		Action: func(c *cli.Context) error {
			s, err := newCommandSigner()
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
