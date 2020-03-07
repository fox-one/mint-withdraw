package mint

var (
	nodes = []string{
		"mixin-node-01.b1.run:8239",
		"mixin-node-02.b1.run:8239",
		"mixin-node-03.b1.run:8239",
		"mixin-node-04.b1.run:8239",
		"mixin-node-05.b1.run:8239",
		"mixin-node-06.b1.run:8239",
		"mixin-node-07.b1.run:8239",
		"node-candy.f1ex.io:8239",
		"node-42.f1ex.io:8239",
		"node-fes.f1ex.io:8239",
	}
)

// SetNodes set mixin nodes
func SetNodes(ns []string) {
	nodes = ns
}
