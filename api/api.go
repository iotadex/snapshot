package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"snapshot/config"
	"snapshot/shimmer"
	"strings"
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
	NftId              string  `json:"nft_id"`
	OutputID           string  `json:"output_id"`
	Name               string  `json:"name"`
	AirDropRewardLevel float64 `json:"airdroprewardlevel"`
	SoonAddr           string  `json:"soon_addr"`
}

type Score struct {
	Level  float64 `json:"airdroprewardlevel"`
	Amount uint64  `json:"amount"`
}

func SnapshotNfts(totalAmount uint64) float64 {
	csvFile, err := os.Open("exp.csv")
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()
	reader := csv.NewReader(csvFile)
	soonSmrAddrs := make(map[string]bool)
	for line, err := reader.Read(); err != io.EOF; line, err = reader.Read() {
		if line[1] == "smr" {
			soonSmrAddrs[line[0]] = true
		}
	}

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
	scores := make(map[string]*Score)
	totalScore := 0.0
	for i, nft := range totalNfts {
		owner := getNftAddressByCondition(nft.Conditions)
		if owner == "smr1qquvmx6m540nemf4h6ajky9f992sx5z5mfydv22u8dgks3wchnzwu6ftp4x" {
			continue
		}
		output := NftOutput{
			Owner:    owner,
			NftId:    nft.NFTID.String(),
			OutputID: totalIds[i],
		}
		if _, exist := soonSmrAddrs[owner]; exist {
			output.SoonAddr = owner
			output.Owner, err = GetRealOwner(nft.NFTID.String())
			if err != nil {
				fmt.Printf("Get real owner error. %s\n", err)
				continue
			}
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
		if _, exist := scores[output.Owner]; exist {
			scores[output.Owner].Level += output.AirDropRewardLevel
		} else {
			scores[output.Owner] = &Score{Level: output.AirDropRewardLevel}
		}
		totalScore += output.AirDropRewardLevel
	}
	for _, s := range scores {
		s.Amount = uint64(s.Level * float64(totalAmount) / totalScore)
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

	jsonBytes, _ = json.Marshal(scores)
	file, err = os.Create("./snaps/scores_" + time.Now().Format("20060102150405") + ".json")
	if err != nil {
		log.Println("Create file error. " + err.Error())
	}
	if _, err = file.Write(jsonBytes); err != nil {
		log.Printf("Write to file error. %v\n", err)
		log.Println(string(jsonBytes))
	} else {
		log.Println("Snapshot all the nfts.")
	}
	return totalScore
}

func getNftAddressByCondition(conds iotago.UnlockConditions) string {
	for _, cond := range conds {
		addr := cond.(*iotago.AddressUnlockCondition).Address.Bech32(iotago.PrefixShimmer)
		return addr
	}
	return ""
}

var ownerUrl string = "https://api.build5.com/api/getMany?collection=nft&fieldName[]=mintingData.nftId&fieldValue[]="
var smrUrl string = "https://api.build5.com/api/getMany?collection=member&fieldName[]=uid&fieldValue[]="

type SoonNFT struct {
	Owner string `json:"owner"`
}
type SoonAccount struct {
	ValidatedAddress map[string]string `json:"validatedAddress"`
}

func GetRealOwner(nftid string) (string, error) {
	data, err := HttpRequest(ownerUrl+nftid, "GET", "", nil)
	if err != nil {
		return "", err
	}
	soonNFT := make([]SoonNFT, 0)
	if err = json.Unmarshal(data, &soonNFT); err != nil {
		return "", err
	}
	if len(soonNFT) < 1 {
		return "", fmt.Errorf("get soonNFT null")
	}
	if data, err = HttpRequest(smrUrl+soonNFT[0].Owner, "GET", "", nil); err != nil {
		return "", err
	}
	smrAdd := make([]SoonAccount, 0)
	if err = json.Unmarshal(data, &smrAdd); err != nil {
		return "", err
	}
	if len(smrAdd) < 1 {
		return "", fmt.Errorf("get smrAdd null")
	}
	return smrAdd[0].ValidatedAddress["smr"], nil
}

func HttpRequest(url string, method string, postParams string, headers map[string]string) ([]byte, error) {
	httpClient := &http.Client{}

	var reader io.Reader
	if len(postParams) > 0 {
		reader = strings.NewReader(postParams)
		if headers == nil {
			headers = map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
		}
	} else {
		reader = nil
	}

	request, err := http.NewRequest(method, url, reader)
	if nil != err {
		return nil, err
	}

	for key, value := range headers {
		request.Header.Add(key, value)
	}

	response, err := httpClient.Do(request)
	if nil != err {
		return nil, err
	}
	defer response.Body.Close()

	return io.ReadAll(response.Body)
}
