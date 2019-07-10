package mint

import (
	"github.com/MixinNetwork/mixin/common"
	jsoniter "github.com/json-iterator/go"
)

// ListMintDistributions list mint distributions
func ListMintDistributions(since, count uint64, showTransaction bool) ([]common.MintDistribution, error) {
	data, err := callRPC("listmintdistributions", []interface{}{since, count, showTransaction})
	if err != nil {
		return nil, err
	}
	dists := []common.MintDistribution{}
	if err := jsoniter.Unmarshal(data, &dists); err != nil {
		return nil, err
	}
	return dists, nil
}
