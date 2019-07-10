package mint

import "github.com/MixinNetwork/mixin/common"

// Signer signer
type Signer interface {
	VerifyOutputs(t *Transaction) ([]int, error)
	Sign(out *common.Transaction, t *Transaction) (*common.VersionedTransaction, error)
}
