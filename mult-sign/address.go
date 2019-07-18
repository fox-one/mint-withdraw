package main

import (
	"encoding/hex"

	"github.com/MixinNetwork/mixin/crypto"
	"github.com/btcsuite/btcutil/base58"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func encodeAddress(c *cli.Context) error {
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

	view := spendPub.DeterministicHashDerive()
	viewPub := view.Public()

	log.Println("view private", view)
	log.Println("view public", viewPub)
	log.Println("address", addressFunc(*spendPub, viewPub))
	return nil
}
