package main

import (
	"fmt"
	"io"
	"os"

	"github.com/fox-one/mint-withdraw/store"
	"github.com/fox-one/mixin-sdk-go/v2"
)

func newSigner(configFile string, rawConfig io.Reader, spendIndex uint64) (*signer, error) {
	if err := setupConfigs(configFile, rawConfig, spendIndex); err != nil {
		return nil, err
	}

	signer := signer{
		walletID: config.WalletID,
	}
	if signer.walletID == "" {
		signer.walletID = config.ClientID
	}

	k, err := NewKey(config.Spends)
	if err != nil {
		return nil, err
	}
	signer.key = k

	for idx, account := range signer.key.Accounts() {
		fmt.Printf("address[%d] %s\n", idx+1, account.String())
	}

	dirPath := fmt.Sprintf(".cache/%v/", signer.key.CacheID())
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// Directory does not exist, so create it
		err := os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			fmt.Println("Error creating directory:", err)
			return nil, err
		}
		fmt.Println("Directory created successfully")
	}

	s, err := store.NewStore(dirPath)
	if err != nil {
		return nil, err
	}
	signer.store = s

	if config.ClientID != "" && config.SessionID != "" && config.SessionKey != "" {
		u, err := mixin.NewFromKeystore(&mixin.Keystore{
			ClientID:   config.ClientID,
			SessionID:  config.SessionID,
			PrivateKey: config.SessionKey,
		})
		if err != nil {
			return nil, err
		}
		signer.user = u
	}

	return &signer, nil
}
