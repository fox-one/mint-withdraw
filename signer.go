package mint

import (
	"context"

	"github.com/MixinNetwork/mixin/common"
)

// Signer signer
type Signer interface {
	Accounts() []*common.Address
	VerifyOutputs(ctx context.Context, store Store, t *Transaction) ([]int, error)
	Sign(out *common.Transaction, t *Transaction) (*common.VersionedTransaction, error)
}
