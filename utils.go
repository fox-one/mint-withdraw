package mint

import (
	"math/rand"
	"time"
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
	return nodes[RandInt(0, len(nodes))]
}

func RandInt(min int, max int) int {
	return min + rand.Intn(max-min)
}
