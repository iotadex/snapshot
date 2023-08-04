package main

import (
	"encoding/hex"
	"fmt"
	"snapshot/api"
	"snapshot/config"
	"snapshot/shimmer"

	iotago "github.com/iotaledger/iota.go/v3"
)

func main() {
	api.SnapshotNfts()
	return
	_, data, _ := iotago.ParseBech32("smr1zr8s7kv070hr0zcrjp40fhjgqv9uvzpgx80u7emnp0ncpgchmxpx25paqmf")
	fmt.Println(hex.EncodeToString((data.(*iotago.NFTAddress))[:]))

	c := shimmer.NewIndexerClient(config.Rpc)
	nft, err := c.GetNftByID("0x3ba971dbb7bfd6d466835a0c8463169e2b41ad7da26ec7dfcfd77140d0eff4c9")
	if err != nil {
		panic(err)
	}
	fmt.Println(*nft)

	nfts, _, err := c.GetNftsByIssuer("smr1zr8s7kv070hr0zcrjp40fhjgqv9uvzpgx80u7emnp0ncpgchmxpx25paqmf")
	if err != nil {
		panic(err)
	}
	fmt.Println(len(nfts))
}
