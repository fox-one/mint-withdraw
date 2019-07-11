package mint

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/MixinNetwork/mixin/common"
	"github.com/MixinNetwork/mixin/crypto"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
)

// Transaction transaction
type Transaction struct {
	common.VersionedTransaction

	Snapshot crypto.Hash `json:"snapshot"`
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
func MakeOutTransaction(t *Transaction, indexs []int, outputAddress, outputAccount string, seed []byte) (*common.Transaction, error) {
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

	tx.Extra = []byte(outputAccount)

	addr, err := common.NewAddressFromString(outputAddress)
	if err != nil {
		return nil, err
	}

	hash := crypto.NewHash(seed)
	seed = append(hash[:], hash[:]...)
	tx.AddOutputWithType(0, []common.Address{addr}, script, amount, seed)
	return tx, nil
}

// ReadTransaction read transaction
func ReadTransaction(hash string) (*Transaction, error) {
	data, err := callRPC("gettransaction", []interface{}{hash})
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
func SendTransaction(raw string) (crypto.Hash, error) {
	data, err := callRPC("sendrawtransaction", []interface{}{raw})
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

// WithdrawTransaction withdraw transaction
func WithdrawTransaction(ctx context.Context, t *Transaction, signer Signer, store Store, addr, extra string) (*Transaction, error) {
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

		out, err := MakeOutTransaction(t, indexs, addr, extra, seed)
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

	for {
		h, err := SendTransaction(rawData)
		if err != nil {
			log.Errorln("send transaction", err)
			time.Sleep(time.Second)
			continue
		}

		t, err := ReadTransaction(h.String())
		if err != nil {
			log.Errorln("read transaction", err)
			time.Sleep(time.Second)
			continue
		}

		if t.Snapshot.HasValue() {
			return t, nil
		}
	}
}
