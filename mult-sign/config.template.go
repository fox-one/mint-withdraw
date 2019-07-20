// +build template

package main

const (
	receiver      = "XINBrP8ybJkg7YkmmJ5JP1Tvx8SoghnrCg1Zp42CxoA5y6StNrMi21ec7zksrCkyQ9KZjhrgYZNHzZQExFcfL5XgDneuhfpb"
	receiverExtra = "318df485-02e1-3c10-8ffd-b241d10dcfd3"

	spendPub = ""
	view     = ""

	sigKey = `-----BEGIN RSA PRIVATE KEY-----
xxx
-----END RSA PRIVATE KEY-----`

	cachePath = "./.cache/"
)

var (
	signerAPIBases = []string{
		"http://localhost:9121",
		"http://localhost:9122",
		"http://localhost:9123",
	}
)
