package mint

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/MixinNetwork/mixin/crypto"
	jsoniter "github.com/json-iterator/go"
)

type memoryStore map[string]string

func (m memoryStore) ReadProperty(_ context.Context, key string) (string, error) {
	value, ok := m[key]
	if !ok {
		return "", nil
	}
	return value, nil
}

func (m memoryStore) WriteProperty(_ context.Context, key, value string) error {
	m[key] = value
	return nil
}

func TestReadUTXOLockWithCacheUsesCachedLockedUTXO(t *testing.T) {
	txHash := crypto.Sha256Hash([]byte("tx"))
	lockHash := crypto.Sha256Hash([]byte("lock"))
	store := memoryStore{}

	bts, err := jsoniter.Marshal(lockedUTXOCache{LockHash: lockHash})
	if err != nil {
		t.Fatal(err)
	}
	store[lockedUTXOStoreKey(txHash, 3)] = string(bts)

	lock, err := ReadUTXOLockWithCache(context.Background(), store, txHash, 3)
	if err != nil {
		t.Fatal(err)
	}
	if lock != lockHash {
		t.Fatalf("lock hash = %s, want %s", lock, lockHash)
	}
}

func TestWithdrawTransactionCacheKeys(t *testing.T) {
	txHash := crypto.Sha256Hash([]byte("tx"))

	if got, want := transactionStoreKey(txHash), "transaction_"+txHash.String(); got != want {
		t.Fatalf("store key = %q, want %q", got, want)
	}
	if got, want := transactionOutputsKey([]int{0, 2, 3}), "0_2_3"; got != want {
		t.Fatalf("outputs key = %q, want %q", got, want)
	}
}

func TestMoveFailedWithdrawTransactionMovesRawDataOutOfActiveCache(t *testing.T) {
	ctx := context.Background()
	txHash := crypto.Sha256Hash([]byte("tx"))
	store := memoryStore{}
	cache := withdrawTransactionCache{
		"0_2_3": "raw-a",
		"4":     "raw-b",
	}

	bts, err := jsoniter.Marshal(cache)
	if err != nil {
		t.Fatal(err)
	}
	store[transactionStoreKey(txHash)] = string(bts)

	if err := moveFailedWithdrawTransaction(ctx, store, txHash, "0_2_3", "raw-a", cache, errors.New("ERROR "+aggregateVerifyError)); err != nil {
		t.Fatal(err)
	}

	var active withdrawTransactionCache
	if err := jsoniter.Unmarshal([]byte(store[transactionStoreKey(txHash)]), &active); err != nil {
		t.Fatal(err)
	}
	if _, ok := active["0_2_3"]; ok {
		t.Fatal("failed raw transaction remained in active cache")
	}
	if got := active["4"]; got != "raw-b" {
		t.Fatalf("active[4] = %q, want raw-b", got)
	}

	var failed []failedWithdrawTransaction
	if err := jsoniter.Unmarshal([]byte(store[failedTransactionStoreKey(txHash)]), &failed); err != nil {
		t.Fatal(err)
	}
	if len(failed) != 1 {
		t.Fatalf("failed entries = %d, want 1", len(failed))
	}
	if failed[0].Outputs != "0_2_3" || failed[0].Raw != "raw-a" || !strings.Contains(failed[0].Error, aggregateVerifyError) {
		t.Fatalf("failed entry = %+v", failed[0])
	}
}
