package mint

import (
	"time"

	"github.com/fox-one/mixin-sdk/utils"
)

func ensureFunc(f func() error) {
	for {
		if err := f(); err == nil {
			return
		}
		time.Sleep(time.Second)
	}
}

func randomNode() string {
	return nodes[utils.RandInt(0, len(nodes))]
}
