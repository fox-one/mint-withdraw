package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupConfigsReadsRawConfigFromStdin(t *testing.T) {
	spend := strings.Repeat("1", 64)
	raw := `spends: ["` + spend + `"]`

	if err := setupConfigs("", strings.NewReader(raw), 0); err != nil {
		t.Fatal(err)
	}
	if len(config.Spends) != 1 || config.Spends[0] != spend {
		t.Fatalf("Spends = %q, want [%q]", config.Spends, spend)
	}
}

func TestSetupConfigsRejectsUnknownFields(t *testing.T) {
	raw := `{"spends": ["` + strings.Repeat("1", 64) + `"], "unknown": "value"}`

	if err := setupConfigs("", strings.NewReader(raw), 0); err == nil {
		t.Fatal("expected unknown field error")
	}
}

func TestSetupConfigsReturnsMissingFileError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")

	if err := setupConfigs(path, nil, 0); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error = %v, want os.ErrNotExist", err)
	}
}

func TestSetupConfigsPreservesPackagedConfigWithRawSpends(t *testing.T) {
	oldPackagedConfig := packagedConfig
	defer func() { packagedConfig = oldPackagedConfig }()

	spend := strings.Repeat("1", 64)
	packagedConfig = Config{
		ClientID:   "client",
		SessionID:  "session",
		SessionKey: "session-key",
		WalletID:   "wallet",
	}

	if err := setupConfigs("", strings.NewReader(`spends: ["`+spend+`"]`), 0); err != nil {
		t.Fatal(err)
	}
	if len(config.Spends) != 1 || config.Spends[0] != spend {
		t.Fatalf("Spends = %q, want [%q]", config.Spends, spend)
	}
	if config.ClientID != packagedConfig.ClientID ||
		config.SessionID != packagedConfig.SessionID ||
		config.SessionKey != packagedConfig.SessionKey ||
		config.WalletID != packagedConfig.WalletID {
		t.Fatalf("config = %+v, want packaged defaults preserved", config)
	}
}

func TestSetupConfigsSelectsSpendByIndex(t *testing.T) {
	firstSpend := strings.Repeat("1", 64)
	secondSpend := strings.Repeat("2", 64)
	raw := `spends: ["` + firstSpend + `", "` + secondSpend + `"]`

	if err := setupConfigs("", strings.NewReader(raw), 2); err != nil {
		t.Fatal(err)
	}
	if len(config.Spends) != 1 || config.Spends[0] != secondSpend {
		t.Fatalf("Spends = %q, want [%q]", config.Spends, secondSpend)
	}
}

func TestSetupConfigsRejectsOutOfRangeSpendIndex(t *testing.T) {
	raw := `spends: ["` + strings.Repeat("1", 64) + `"]`

	if err := setupConfigs("", strings.NewReader(raw), 2); err == nil {
		t.Fatal("expected out of range spend index error")
	}
}
