package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Spends     []string `yaml:"spends"`
	ClientID   string   `yaml:"client_id"`
	SessionID  string   `yaml:"session_id"`
	SessionKey string   `yaml:"session_key"`
	WalletID   string   `yaml:"wallet_id"`
}

var (
	config         Config
	packagedConfig Config
)

func setupConfigs(configFile string, rawConfig io.Reader, spendIndex uint64) error {
	config = packagedConfig

	configFile = strings.TrimSpace(configFile)
	if rawConfig != nil {
		bts, err := io.ReadAll(rawConfig)
		if err != nil {
			return fmt.Errorf("read stdin config: %w", err)
		}
		if err := decodeConfig(bts, "stdin"); err != nil {
			return err
		}
	} else if configFile != "" {
		if bts, err := os.ReadFile(configFile); err != nil {
			return fmt.Errorf("read config %s: %w", configFile, err)
		} else {
			if err := decodeConfig(bts, configFile); err != nil {
				return err
			}
		}
	}

	config.applyEnv()
	config.trim()

	if len(config.Spends) == 0 {
		return errors.New("missing spend keys: set spends in config yaml or MINT_SPEND_KEYS")
	}
	if spendIndex > 0 {
		if spendIndex > uint64(len(config.Spends)) {
			return fmt.Errorf("spend index %d out of range: have %d spend keys", spendIndex, len(config.Spends))
		}
		config.Spends = []string{config.Spends[spendIndex-1]}
	}

	return nil
}

func decodeConfig(bts []byte, name string) error {
	if len(strings.TrimSpace(string(bts))) == 0 {
		return nil
	}

	decoder := yaml.NewDecoder(bytes.NewReader(bts))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		return fmt.Errorf("decode config %s: %w", name, err)
	}
	return nil
}

func (c *Config) applyEnv() {
	setSpendsFromEnv(&c.Spends, "MINT_SPEND_KEYS", "MINT_SPENDS", "MINT_SPEND_KEY")
	setConfigFromEnv(&c.ClientID, "MINT_CLIENT_ID")
	setConfigFromEnv(&c.SessionID, "MINT_SESSION_ID")
	setConfigFromEnv(&c.SessionKey, "MINT_SESSION_KEY")
	setConfigFromEnv(&c.WalletID, "MINT_WALLET_ID", "MINT_RECEIVER_WALLET_ID", "MINT_RECEIVER_WALLET")
}

func (c *Config) trim() {
	c.Spends = trimStrings(c.Spends)
	c.ClientID = strings.TrimSpace(c.ClientID)
	c.SessionID = strings.TrimSpace(c.SessionID)
	c.SessionKey = strings.TrimSpace(c.SessionKey)
	c.WalletID = strings.TrimSpace(c.WalletID)
}

func setConfigFromEnv(dst *string, names ...string) {
	if strings.TrimSpace(*dst) != "" {
		return
	}

	for _, name := range names {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			*dst = v
			return
		}
	}
}

func setSpendsFromEnv(dst *[]string, names ...string) {
	if len(trimStrings(*dst)) > 0 {
		return
	}

	for _, name := range names {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			*dst = splitSpends(v)
			return
		}
	}
}

func splitSpends(v string) []string {
	return trimStrings(strings.Split(v, ","))
}

func trimStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}
