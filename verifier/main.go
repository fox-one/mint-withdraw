package main

import (
	"crypto/rand"
	"fmt"
	"os"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

type (
	Keystore struct {
		Signer *crypto.Key `json:"s"`
		Payee  *crypto.Key `json:"p"`
	}
)

var (
	commands cli.Commands
)

func main() {
	app := cli.NewApp()
	app.Name = "verifier"
	app.Version = "1.0.0"

	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "debug"},
	}

	app.Before = func(c *cli.Context) error {
		if c.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}

		return nil
	}

	app.Commands = append(app.Commands, commands...)

	if err := app.Run(os.Args); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}

func createAddress() (*common.Address, error) {
	var (
		seed   = make([]byte, 64)
		length = 0
	)

	for length < len(seed) {
		n, err := rand.Read(seed)
		if err != nil {
			return nil, err
		}
		length += n
	}

	s := crypto.NewKeyFromSeed(seed)
	S := s.Public()
	v := S.DeterministicHashDerive()
	V := v.Public()
	return &common.Address{
		PrivateSpendKey: s,
		PublicSpendKey:  S,
		PrivateViewKey:  v,
		PublicViewKey:   V,
	}, nil
}

func printAddress(addr common.Address) {
	fmt.Println("Address", addr.String())
	fmt.Println("PrivateSpendKey", addr.PrivateSpendKey.String())
	fmt.Println("PublicSpendKey", addr.PublicSpendKey.String())
	fmt.Println("PrivateViewKey", addr.PrivateViewKey.String())
	fmt.Println("PublicViewKey", addr.PublicViewKey.String())
}
