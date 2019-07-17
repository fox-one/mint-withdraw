// +build template

package main

import "github.com/MixinNetwork/mixin/common"

const (
	Port = 9121

	View          = ""
	Spend         = ""
	CoSignerCount = 3

	SigKey = `-----BEGIN PUBLIC KEY-----
xxx
-----END PUBLIC KEY-----`
)

var (
	acceptedOutputTypes = map[uint8]bool{
		common.OutputTypeScript:     true,
		common.OutputTypeNodePledge: true,
	}
)
