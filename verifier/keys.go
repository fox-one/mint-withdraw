package main

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli"
)

func init() {
	commands = append(commands, cli.Command{
		Name: "keys",
		Action: func(c *cli.Context) error {
			var key Keystore
			{
				signer, err := createAddress()
				if err != nil {
					return err
				}
				fmt.Println("Signer")
				printAddress(*signer)
				fmt.Println()
				key.Signer = &signer.PrivateSpendKey
			}

			{
				payee, err := createAddress()
				if err != nil {
					return err
				}
				fmt.Println("Payee")
				printAddress(*payee)
				fmt.Println()
				key.Payee = &payee.PrivateSpendKey
			}

			{
				fmt.Println("keystore")
				bts, _ := json.MarshalIndent(key, "", "    ")
				fmt.Println(string(bts))
			}

			return nil
		},
	})
}
