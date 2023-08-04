package api

import (
	"encoding/json"
	"log"
	"os"
	"snapshot/config"
	"snapshot/shimmer"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	iotago "github.com/iotaledger/iota.go/v3"
)

func Snapshot(c *gin.Context) {

}

type NftFeat struct {
	Name       string           `json:"name"`
	Attributes []AttributeTrait `json:"attributes"`
}

type AttributeTrait struct {
	TraitType string  `json:"trait_type"`
	Value     float64 `json:"value,string"`
}

type NftOutput struct {
	Owner              string  `json:"owner"`
	OutputID           string  `json:"output_id"`
	Name               string  `json:"name"`
	AirDropRewardLevel float64 `json:"airdroprewardlevel"`
}

func SnapshotNfts() {
	client := shimmer.NewIndexerClient(config.Rpc)
	var wg sync.WaitGroup
	var totalNfts []*iotago.NFTOutput
	var totalIds []string
	var mu sync.Mutex
	for _, nftAddr := range config.NftAddresses {
		wg.Add(1)
		go func(addr string) {
			nfts, ids, err := client.GetNftsByIssuer(addr)
			if err != nil {
				log.Panicf("GetNftsByIssuer error. %v", err)
			}
			mu.Lock()
			totalNfts = append(totalNfts, nfts...)
			totalIds = append(totalIds, ids...)
			mu.Unlock()
			wg.Done()
		}(nftAddr)
	}
	wg.Wait()
	nftOutputs := make([]NftOutput, 0)
	for i, nft := range totalNfts {
		output := NftOutput{
			Owner:    getNftAddressByCondition(nft.Conditions),
			OutputID: totalIds[i],
		}
		for _, feat := range nft.ImmutableFeatures {
			if feat.Type() != iotago.FeatureMetadata {
				continue
			}
			meta := feat.(*iotago.MetadataFeature)
			nftFeat := NftFeat{}
			if err := json.Unmarshal(meta.Data, &nftFeat); err != nil {
				log.Panicf("Unmarshal nftFeat error. %v", err)
			}
			for i := range nftFeat.Attributes {
				if nftFeat.Attributes[i].TraitType == "airdroprewardlevel" {
					output.Name = nftFeat.Name
					output.AirDropRewardLevel = nftFeat.Attributes[i].Value
					break
				}
			}
		}
		nftOutputs = append(nftOutputs, output)
	}
	jsonBytes, _ := json.Marshal(nftOutputs)
	if err := os.MkdirAll("./snaps", os.ModePerm); err != nil {
		log.Println("Create dir './snaps' error. " + err.Error())
	}
	file, err := os.Create("./snaps/" + time.Now().Format("20060102150405") + ".json")
	if err != nil {
		log.Println("Create file error. " + err.Error())
	}
	if _, err = file.Write(jsonBytes); err != nil {
		log.Printf("Write to file error. %v\n", err)
		log.Println(string(jsonBytes))
	} else {
		log.Println("Snapshot all the nfts.")
	}
}

func getNftAddressByCondition(conds iotago.UnlockConditions) string {
	for _, cond := range conds {
		addr := cond.(*iotago.AddressUnlockCondition).Address.Bech32(iotago.PrefixShimmer)
		return addr
	}
	return ""
}
