package mint

var (
	legacyNodes = []string{
		"node-candy.f1ex.io:8239",
		"node-42.f1ex.io:8239",
		"node-box.f1ex.io:8239",
		"node-box-2.f1ex.io:8239",
	}

	freshNodes = []string{
		"mixin-node-42.f1ex.io:8239",
		"mixin-node-box-1.b.watch:8239",
		"mixin-node-box-2.b.watch:8239",
		"mixin-node-box-3.b.watch:8239",
		"mixin-node-box-4.b.watch:8239",
	}

	nodes = []string{"https://kernel.mixin.dev:443"}
)

// SetNodes set mixin nodes
func SetNodes(ns []string) {
	nodes = ns
}
