package mint

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

// Transaction transaction
type Transaction struct {
	common.VersionedTransaction

	Snapshot string      `json:"snapshot"`
	Hash     crypto.Hash `json:"hash,omitempty"`
}

type UTXO struct {
	common.Input
	common.Output
	Lock crypto.Hash `json:"lock,omitempty"`
}

// ReadUTXO read utxo
func ReadUTXO(hash crypto.Hash, index int, node ...string) (*common.UTXOWithLock, error) {
	var n = RandomNode()
	if len(node) > 0 && node[0] != "" {
		n = node[0]
	}

	data, err := callRPC(n, "getutxo", []interface{}{hash, index})
	if err != nil {
		return nil, err
	}

	var o UTXO
	if err := jsoniter.Unmarshal(data, &o); err != nil {
		return nil, err
	}
	out := &common.UTXOWithLock{
		LockHash: o.Lock,
	}
	out.UTXO.Input = o.Input
	out.UTXO.Output = o.Output
	return out, nil
}

type lockedUTXOCache struct {
	LockHash crypto.Hash `json:"lock_hash"`
}

type withdrawTransactionCache map[string]string

type failedWithdrawTransaction struct {
	Outputs  string    `json:"outputs"`
	Raw      string    `json:"raw"`
	Error    string    `json:"error"`
	FailedAt time.Time `json:"failed_at"`
}

const aggregateVerifyError = "AggregateVerify signature verify failed"

// ReadUTXOLockWithCache caches only non-empty locks because unlocked outputs must be checked live.
func ReadUTXOLockWithCache(ctx context.Context, store Store, hash crypto.Hash, index int, node ...string) (crypto.Hash, error) {
	storeKey := lockedUTXOStoreKey(hash, index)
	if store != nil {
		if cached, err := store.ReadProperty(ctx, storeKey); err == nil && strings.TrimSpace(cached) != "" {
			var cachedUTXO lockedUTXOCache
			if err := jsoniter.Unmarshal([]byte(cached), &cachedUTXO); err == nil && cachedUTXO.LockHash.HasValue() {
				return cachedUTXO.LockHash, nil
			}
		}
	}

	utxo, err := ReadUTXO(hash, index, node...)
	if err != nil {
		return crypto.Hash{}, err
	}
	if store != nil && utxo.LockHash.HasValue() {
		cachedUTXO := lockedUTXOCache{LockHash: utxo.LockHash}
		if bts, err := jsoniter.Marshal(cachedUTXO); err == nil {
			if err := store.WriteProperty(ctx, storeKey, string(bts)); err != nil {
				log.WithError(err).Warnln("cache locked utxo", hash.String(), index)
			}
		}
	}
	return utxo.LockHash, nil
}

func (t Transaction) ReadUTXOKeys(hash crypto.Hash, index uint) (*common.UTXOKeys, error) {
	if t.Hash.String() != hash.String() {
		return nil, errors.New("hash not matched")
	}
	if index >= uint(len(t.Outputs)) {
		return nil, errors.New("index exceeds output bounds")
	}
	o := t.Outputs[index]
	out := &common.UTXOKeys{}
	out.Keys = o.Keys
	out.Mask = o.Mask
	return out, nil
}

// CheckDepositInput check deposit
func (t Transaction) CheckDepositInput(deposit *common.DepositData, tx crypto.Hash) error {
	return nil
}

// ReadLastMintDistribution read last mint distribution
func (t Transaction) ReadLastMintDistribution(group string) (*common.MintDistribution, error) {
	return nil, nil
}

// MakeOutTransaction make out transaction
func MakeOutTransaction(t *Transaction, indexs []int, mask crypto.Key, keys []*crypto.Key, extra string) (*common.Transaction, error) {
	if len(indexs) == 0 {
		return nil, nil
	}

	tx := common.NewTransactionV5(t.Asset)

	amount := common.NewInteger(0)
	var script common.Script
	for _, i := range indexs {
		if i >= len(t.Outputs) {
			return nil, errors.New("index exceeds output bounds")
		}

		o := t.Outputs[i]
		script = o.Script
		amount = amount.Add(o.Amount)
		tx.AddInput(t.Hash, uint(i))
	}

	tx.Extra = []byte(extra)
	tx.Outputs = []*common.Output{
		{
			Type:   common.OutputTypeScript,
			Amount: amount,
			Keys:   keys,
			Script: script,
			Mask:   mask,
		},
	}
	return tx, nil
}

// ReadTransaction read transaction
func ReadTransaction(hash string, node ...string) (*Transaction, error) {
	var n = RandomNode()
	if len(node) > 0 && node[0] != "" {
		n = node[0]
	}

	data, err := callRPC(n, "gettransaction", []interface{}{hash})
	if err != nil {
		return nil, err
	}
	var t struct {
		Transaction
		Extra string `json:"extra"`
	}
	if err := jsoniter.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	if bts, err := hex.DecodeString(t.Extra); err == nil {
		t.Transaction.Extra = bts
	}
	if !t.Hash.HasValue() {
		return nil, errors.New("null transaction")
	}
	return &t.Transaction, nil
}

// SendTransaction send transaction
func SendTransaction(raw string, node ...string) (crypto.Hash, error) {
	var n = RandomNode()
	if len(node) > 0 && node[0] != "" {
		n = node[0]
	}

	data, err := callRPC(n, "sendrawtransaction", []interface{}{raw})
	if err != nil {
		return crypto.Hash{}, err
	}
	var resp struct {
		Hash crypto.Hash `json:"hash"`
	}
	if err := jsoniter.Unmarshal(data, &resp); err != nil {
		return crypto.Hash{}, err
	}
	return resp.Hash, nil
}

// DoTransaction do transaction
func DoTransaction(ctx context.Context, rawData string) (*Transaction, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("do transaction not done")

		case <-ticker.C:
		}

		node := RandomNode()
		h, err := SendTransaction(rawData, node)
		if err != nil {
			if strings.HasPrefix(err.Error(), "ERROR invalid output key ") {
				return nil, nil
			}

			if strings.HasPrefix(err.Error(), "ERROR invalid lock ") {
				return nil, err
			}

			if strings.Contains(err.Error(), aggregateVerifyError) {
				return nil, err
			}

			log.Errorln("send transaction", err, rawData)
			continue
		}

		log.Info("output transaction hash: ", h)
		for i := 0; i < 6; i++ {
			t, err := ReadTransaction(h.String(), node)
			if err != nil {
				log.Errorln("read transaction", err)
				continue
			}

			if _, err := crypto.HashFromString(t.Snapshot); err == nil {
				return t, nil
			}
		}
	}
}

// WithdrawTransaction withdraw transaction
func WithdrawTransaction(ctx context.Context, t *Transaction, signer Signer, store Store, mask crypto.Key, keys []*crypto.Key, extra string) (*Transaction, error) {
	var rawData = ""

	log.Infoln("try withdraw", "signer: ", signer.Accounts(), "tx: ", t.Hash.String())
	indexs, err := signer.VerifyOutputs(ctx, store, t)
	if err != nil || len(indexs) == 0 {
		return nil, err
	}

	storeKey := transactionStoreKey(t.Hash)
	outputsKey := transactionOutputsKey(indexs)
	cachedTransactions := withdrawTransactionCache{}
	ensureFunc(func() error {
		v, e := store.ReadProperty(ctx, storeKey)
		if e != nil {
			log.Errorln("read property", storeKey, e)
			return e
		}
		if strings.TrimSpace(v) == "" {
			return nil
		}
		if err := jsoniter.Unmarshal([]byte(v), &cachedTransactions); err != nil {
			log.WithError(err).Warnln("read transaction cache", storeKey)
			cachedTransactions = withdrawTransactionCache{}
			return nil
		}
		rawData = cachedTransactions[outputsKey]
		return nil
	})

	if rawData == "" {
		var seed = make([]byte, 64)
		ensureFunc(func() error {
			_, err := rand.Read(seed)
			if err == nil {
				return nil
			}
			log.Errorln("rand read", err)
			time.Sleep(time.Second)
			return err
		})
		out, err := MakeOutTransaction(t, indexs, mask, keys, extra)
		if err != nil {
			log.WithError(err).Errorln("make out transaction")
			return nil, err
		}

		signed, err := signer.Sign(out, t)
		if err != nil {
			log.WithError(err).Errorln("sign transaction")
			return nil, err
		}
		rawData = hex.EncodeToString(signed.Marshal())

		cachedTransactions[outputsKey] = rawData
		ensureFunc(func() error {
			bts, err := jsoniter.Marshal(cachedTransactions)
			if err != nil {
				log.Errorln("marshal transaction cache", storeKey, err)
				return err
			}
			e := store.WriteProperty(ctx, storeKey, string(bts))
			if e == nil {
				return nil
			}
			log.Errorln("write property", storeKey, e)
			return e
		})
	}

	tx, err := DoTransaction(ctx, rawData)
	if err != nil {
		if strings.Contains(err.Error(), aggregateVerifyError) {
			txErr := err
			log.Errorln(err)
			ensureFunc(func() error {
				if err := moveFailedWithdrawTransaction(ctx, store, t.Hash, outputsKey, rawData, cachedTransactions, txErr); err != nil {
					log.WithError(err).Errorln("move failed transaction cache", storeKey)
					return err
				}
				return nil
			})
		}
	}

	return tx, err
}

func transactionStoreKey(hash crypto.Hash) string {
	return fmt.Sprintf("transaction_%s", hash.String())
}

func failedTransactionStoreKey(hash crypto.Hash) string {
	return fmt.Sprintf("transaction_failed_%s", hash.String())
}

func transactionOutputsKey(indexs []int) string {
	parts := make([]string, 0, len(indexs))
	for _, index := range indexs {
		parts = append(parts, strconv.Itoa(index))
	}
	return strings.Join(parts, "_")
}

func lockedUTXOStoreKey(hash crypto.Hash, index int) string {
	return fmt.Sprintf("utxo_lock_%s_%d", hash.String(), index)
}

func moveFailedWithdrawTransaction(ctx context.Context, store Store, hash crypto.Hash, outputsKey, rawData string, cache withdrawTransactionCache, cause error) error {
	if store == nil || rawData == "" {
		return nil
	}

	if err := appendFailedWithdrawTransaction(ctx, store, hash, outputsKey, rawData, cause); err != nil {
		return err
	}

	delete(cache, outputsKey)
	bts, err := jsoniter.Marshal(cache)
	if err != nil {
		return err
	}
	return store.WriteProperty(ctx, transactionStoreKey(hash), string(bts))
}

func appendFailedWithdrawTransaction(ctx context.Context, store Store, hash crypto.Hash, outputsKey, rawData string, cause error) error {
	storeKey := failedTransactionStoreKey(hash)
	var failures []failedWithdrawTransaction
	if cached, err := store.ReadProperty(ctx, storeKey); err != nil {
		return err
	} else if strings.TrimSpace(cached) != "" {
		if err := jsoniter.Unmarshal([]byte(cached), &failures); err != nil {
			return err
		}
	}

	causeText := ""
	if cause != nil {
		causeText = cause.Error()
	}
	for _, failure := range failures {
		if failure.Outputs == outputsKey && failure.Raw == rawData && failure.Error == causeText {
			return nil
		}
	}

	failures = append(failures, failedWithdrawTransaction{
		Outputs:  outputsKey,
		Raw:      rawData,
		Error:    causeText,
		FailedAt: time.Now(),
	})
	bts, err := jsoniter.Marshal(failures)
	if err != nil {
		return err
	}
	return store.WriteProperty(ctx, storeKey, string(bts))
}
