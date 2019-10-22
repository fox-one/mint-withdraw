package mint

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
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
	Hash     crypto.Hash `json:"hash"`
}

// ReadUTXO read utxo
func (t Transaction) ReadUTXO(hash crypto.Hash, index int) (*common.UTXOWithLock, error) {
	if t.Hash.String() != hash.String() {
		return nil, errors.New("hash not matched")
	}
	if index >= len(t.Outputs) {
		return nil, errors.New("index exceeds output bounds")
	}
	o := t.Outputs[index]
	out := &common.UTXOWithLock{}
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
func MakeOutTransaction(t *Transaction, indexs []int, outputAddress string, mask crypto.Key, keys []crypto.Key, extra string) (*common.Transaction, error) {
	if len(indexs) == 0 {
		return nil, nil
	}

	tx := common.NewTransaction(t.Asset)

	amount := common.NewInteger(0)
	var script common.Script
	for _, i := range indexs {
		if i >= len(t.Outputs) {
			return nil, errors.New("index exceeds output bounds")
		}

		o := t.Outputs[i]
		script = o.Script
		amount = amount.Add(o.Amount)
		tx.AddInput(t.Hash, i)
	}

	tx.Extra = []byte(extra)

	if len(outputAddress) > 0 {
		addr, err := common.NewAddressFromString(outputAddress)
		if err != nil {
			return nil, err
		}

		tx.AddRandomScriptOutput([]common.Address{addr}, script, amount)
	} else {
		tx.Outputs = []*common.Output{
			&common.Output{
				Type:   common.OutputTypeScript,
				Amount: amount,
				Keys:   keys,
				Script: script,
				Mask:   mask,
			},
		}
	}
	return tx, nil
}

// ReadTransaction read transaction
func ReadTransaction(hash string, node ...string) (*Transaction, error) {
	var n = randomNode()
	if len(node) > 0 && node[0] != "" {
		n = node[0]
	}

	data, err := callRPC(n, "gettransaction", []interface{}{hash})
	if err != nil {
		return nil, err
	}
	t := Transaction{}
	if err := jsoniter.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// SendTransaction send transaction
func SendTransaction(raw string, node ...string) (crypto.Hash, error) {
	var n = randomNode()
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
	for {
		node := randomNode()
		h, err := SendTransaction(rawData, node)
		if err != nil {
			prefix := "ERROR invalid output key "
			if strings.HasPrefix(err.Error(), prefix) {
				return nil, nil
			}

			log.Errorln("send transaction", err)
			time.Sleep(time.Second)
			continue
		}

		log.Info("output transaction hash: ", h)
		for i := 0; i < 6; i++ {
			t, err := ReadTransaction(h.String(), node)
			if err != nil {
				log.Errorln("read transaction", err)
				time.Sleep(time.Second)
				continue
			}

			if _, err := crypto.HashFromString(t.Snapshot); err == nil {
				return t, nil
			}

			time.Sleep(time.Second)
		}
	}
}

// WithdrawTransaction withdraw transaction
func WithdrawTransaction(ctx context.Context, t *Transaction, signer Signer, store Store, addr string, mask crypto.Key, keys []crypto.Key, extra string) (*Transaction, error) {
	var rawData = ""

	storeKey := fmt.Sprintf("transaction_%s", t.Hash.String())
	ensureFunc(func() error {
		v, e := store.ReadProperty(ctx, storeKey)
		if e == nil {
			rawData = v
			return nil
		}
		log.Errorln("read property", storeKey, e)
		return e
	})

	if rawData == "" {
		indexs, err := signer.VerifyOutputs(t)
		if err != nil || len(indexs) == 0 {
			return nil, err
		}

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
		out, err := MakeOutTransaction(t, indexs, addr, mask, keys, extra)
		if err != nil {
			return nil, err
		}

		signed, err := signer.Sign(out, t)
		if err != nil {
			return nil, err
		}
		rawData = hex.EncodeToString(signed.Marshal())

		ensureFunc(func() error {
			e := store.WriteProperty(ctx, storeKey, rawData)
			if e == nil {
				return nil
			}
			log.Errorln("write property", storeKey, e)
			return e
		})
	}

	return DoTransaction(ctx, rawData)
}
