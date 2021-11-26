package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/MixinNetwork/mixin/common"
	"github.com/urfave/cli"
)

func init() {
	commands = append(commands, cli.Command{
		Name: "verify",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "keystore, k"},
			cli.StringFlag{Name: "raw"},
		},
		Action: func(c *cli.Context) error {
			key, err := loadKeystore(c.String("keystore"))
			if err != nil {
				return err
			}

			raw := c.String("raw")
			rawData, err := hex.DecodeString(raw)
			if err != nil {
				return err
			}

			tx, err := common.UnmarshalVersionedTransaction(rawData)
			if err != nil {
				return err
			}

			var (
				extras [64]byte
				S      = key.Signer.Public()
				P      = key.Payee.Public()
			)
			copy(extras[:32], S[:])
			copy(extras[32:], P[:])

			if bytes.Compare(tx.Extra, extras[:]) != 0 {
				return fmt.Errorf("expected: %s; got: %s", hex.EncodeToString(extras[:]), hex.EncodeToString(tx.Extra[:]))
			}

			fmt.Println("verified")
			return nil
		},
	})
}

func loadKeystore(keystore string) (*Keystore, error) {
	bts, err := ioutil.ReadFile(keystore)
	if err != nil {
		return nil, err
	}
	var key Keystore
	if err := json.Unmarshal(bts, &key); err != nil {
		return nil, err
	}
	if key.Signer == nil || !key.Signer.HasValue() ||
		key.Payee == nil || !key.Payee.HasValue() {
		return nil, errors.New("unmarshal keystore failed")
	}
	return &key, nil
}
