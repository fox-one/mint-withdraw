// +build template

package main

const (
	Receiver      = "XINBrP8ybJkg7YkmmJ5JP1Tvx8SoghnrCg1Zp42CxoA5y6StNrMi21ec7zksrCkyQ9KZjhrgYZNHzZQExFcfL5XgDneuhfpb"
	ReceiverExtra = "318df485-02e1-3c10-8ffd-b241d10dcfd3"

	transaction = "7d5b5a38be50ca196eebc5057e1bd98655f8e8e6ea356c112233da9e1c556dc3"
)

var (
	signerAPIBases = []string{
		"http://localhost:9121",
		"http://localhost:9122",
		"http://localhost:9123",
	}
)
