package mint

import (
	"github.com/MixinNetwork/mixin/common"
	jsoniter "github.com/json-iterator/go"
)

// ListMintDistributions list mint distributions
func ListMintDistributions(since, count uint64, node ...string) ([]common.MintDistribution, error) {
	var n = randomNode()
	if len(node) > 0 && node[0] != "" {
		n = node[0]
	}

	data, err := callRPC(n, "listmintdistributions", []interface{}{since, count, false})
	if err != nil {
		return nil, err
	}
	dists := []common.MintDistribution{}
	if err := jsoniter.Unmarshal(data, &dists); err != nil {
		return nil, err
	}
	return dists, nil
}
