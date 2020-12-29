package mint

var (
	nodes = []string{
		"node-candy.f1ex.io:8239",
		"node-42.f1ex.io:8239",
		"node-box.f1ex.io:8239",
		"node-box-2.f1ex.io:8239",
	}
)

// SetNodes set mixin nodes
func SetNodes(ns []string) {
	nodes = ns
}
