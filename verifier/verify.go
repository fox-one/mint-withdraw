package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	"github.com/urfave/cli"
)

const (
	XIN = "a99c2e0e2b1da4d648755ef19bd95139acbbe6564cfb06dec7cd34931ca72cdc"
)

func init() {
	commands = append(commands, cli.Command{
		Name: "verify",
		Flags: []cli.Flag{
			cli.StringFlag{Name: "keystore, k"},
			cli.StringFlag{Name: "raw"},
			cli.BoolFlag{Name: "insecure"},
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

			if !bytes.Equal(tx.Extra, extras[:]) {
				return fmt.Errorf("extra expected: %s; got: %s", hex.EncodeToString(extras[:]), hex.EncodeToString(tx.Extra[:]))
			}

			if tx.Outputs[0].Type != common.OutputTypeNodePledge {
				return fmt.Errorf("type expected: %d; got: %d", common.OutputTypeNodePledge, tx.Outputs[0].Type)
			}

			if !c.Bool("insecure") {
				if tx.Asset.String() != XIN {
					return fmt.Errorf("asset expected: %s; got: %s", XIN, tx.Asset.String())
				}

				if tx.Outputs[0].Amount.Cmp(common.NewIntegerFromString("13439")) != 0 {
					return fmt.Errorf("amount expected: 13439; got: %s", tx.Outputs[0].Amount.String())
				}

				payload := tx.PayloadMarshal()

				var pubKeys []*crypto.Key
				for inputIndex, in := range tx.Inputs {
					utxo, err := ReadUTXOLock(in.Hash, in.Index)
					if err != nil {
						return err
					} else if utxo == nil {
						return fmt.Errorf("input (%s:%d) not found", in.Hash, in.Index)
					}

					if utxo.LockHash.HasValue() && utxo.LockHash != tx.PayloadHash() {
						return fmt.Errorf("input (%s:%d) locked by %s", in.Hash, in.Index, utxo.LockHash.String())
					}

					if tx.AggregatedSignature == nil {
						signatues := tx.SignaturesMap[inputIndex]
						verified := 0
						for keyIndex, key := range utxo.Keys {
							if sig, ok := signatues[uint16(keyIndex)]; ok {
								if !key.Verify(payload, *sig) {
									return fmt.Errorf("input (%d) signature (%d) verify failed", inputIndex, keyIndex)
								}
								verified++
							}
						}
						if err := utxo.Script.Validate(verified); err != nil {
							return fmt.Errorf("input (%d) got insufficient signatures (%d)", inputIndex, verified)
						}
					}

					pubKeys = append(pubKeys, utxo.Keys...)
				}

				if tx.AggregatedSignature != nil {
					if err := crypto.AggregateVerify(&tx.AggregatedSignature.Signature, pubKeys, tx.AggregatedSignature.Signers, payload); err != nil {
						return fmt.Errorf("aggregate verify failed")
					}
				} else if len(tx.SignaturesMap) == 0 {
					return fmt.Errorf("empty signatures")
				}
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
