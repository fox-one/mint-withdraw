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
		"mixin-node.matpool.io:8239",
		"35.188.235.212:8239",
		"mixin-node0.exinpool.com:8239",
		"mixin-node.candy.one:8239",
		"onda.mixin-node.tako.vip:8239",
		"noodle.mixin-node.tako.vip:8239",
		"tako.mixin-node.tako.vip:8239",
		"ss.mixin-node.tako.vip:8239",
		"mixin-node0.eoslaomao.com:1443",
		"mixin-node1.eoslaomao.com:1443",
		"node-42.f1ex.io:1443",
		"35.234.74.25:8239",
		"35.234.96.182:8239",
		"ss2.mixin-node.tako.vip:8239",
		"35.188.242.130:8239",
		"35.245.207.174:8239",
		"35.227.72.6:8239",
	}
)

// SetNodes set mixin nodes
func SetNodes(ns []string) {
	nodes = ns
}
