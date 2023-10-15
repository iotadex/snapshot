package config

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"os"

	iotago "github.com/iotaledger/iota.go/v3"
)

var (
	HttpPort     int
	Rpc          string
	NftIds       []string
	NftAddresses []string
)

// Load load config file
func init() {
	file, err := os.Open("config/config.json")
	if err != nil {
		log.Panic(err)
	}
	defer file.Close()
	type Config struct {
		HttpPort int      `json:"http_port"`
		Rpc      string   `json:"rpc"`
		NftIds   []string `json:"nft_ids"`
	}
	all := &Config{}
	if err = json.NewDecoder(file).Decode(all); err != nil {
		log.Panic(err)
	}
	HttpPort = all.HttpPort
	Rpc = all.Rpc
	NftIds = all.NftIds

	for _, id := range NftIds {
		if len(id) >= 2 && id[0] == '0' && (id[1] == 'x' || id[1] == 'X') {
			id = id[2:]
		}
		data, _ := hex.DecodeString(id)
		if len(data) < iotago.NFTIDLength {
			log.Panicf("error nft id. %s", id)
		}
		var addr iotago.NFTAddress
		copy(addr[:], data)
		NftAddresses = append(NftAddresses, addr.Bech32(iotago.PrefixShimmer))
	}
}
