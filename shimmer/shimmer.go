package shimmer

import (
	"context"
	"encoding/hex"
	"fmt"

	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

type IndexerClient struct {
	url string
}

func NewIndexerClient(_url string) *IndexerClient {
	return &IndexerClient{
		url: _url,
	}
}

func (ic *IndexerClient) GetNftsByIssuer(issuer string) ([]*iotago.NFTOutput, []string, error) {
	client := nodeclient.New(ic.url)
	indexerClient, err := client.Indexer(context.TODO())
	if err != nil {
		return nil, nil, fmt.Errorf("indexer client got error. %v", err)
	}
	query := nodeclient.NFTsQuery{
		IssuerBech32: issuer,
	}
	res, err := indexerClient.Outputs(context.TODO(), &query)
	if err != nil {
		return nil, nil, fmt.Errorf("nft outputs filter error. %v", err)
	}
	nfts := make([]*iotago.NFTOutput, 0)
	ids := make([]string, 0)
	for res.Next() {
		outputs, _ := res.Outputs()
		for i, output := range outputs {
			if output.Type() == iotago.OutputNFT {
				nfts = append(nfts, output.(*iotago.NFTOutput))
				ids = append(ids, res.Response.Items[i])
			}
		}
	}

	return nfts, ids, nil
}

func (ic *IndexerClient) GetNftByID(id string) (*iotago.NFTOutput, error) {
	client := nodeclient.New(ic.url)
	indexerClient, err := client.Indexer(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("indexer client got error. %v", err)
	}
	if len(id) >= 2 && id[0] == '0' && (id[1] == 'x' || id[1] == 'X') {
		id = id[2:]
	}
	data, _ := hex.DecodeString(id)
	if len(data) < iotago.NFTIDLength {
		return nil, fmt.Errorf("error nft id. %s", id)
	}
	var nftid iotago.NFTID
	copy(nftid[:], data)
	_, nft, _, err := indexerClient.NFT(context.TODO(), nftid)
	if err != nil {
		return nil, fmt.Errorf("nft id query error. %s : %v", id, err)
	}

	return nft, nil
}
