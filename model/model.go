package model

// Asset is the entity (video) which is uploaded by the user.
type Asset struct {
	AssetID         string `json:"AssetID"`
	AssetStatusCode uint32 `json:"AssetStatusCode"`
	AssetStatus     string `json:"AssetStatus"`
	AssetError      bool   `json:"AssetError"`
	StreamURL       string `json:"StreamURL"`
	CreatedAt       int64  `json:"CreatedAt"`
}

// TranscodingDeal is the type binding for a transcoding deal in the livepeer network.
type TranscodingDeal struct {
	AssetID                  string `json:"AssetID"`
	TranscodingCost          string `json:"TranscodingCost"`
	TranscodingCostEstimated string `json:"TranscodingCostEstimated"`
}

// StorageDeal is the type binding for a storage deal in the IPFS/filecoin network.
type StorageDeal struct {
	AssetID              string `json:"AssetID"`
	StorageStatusCode    uint32 `json:"StorageStatusCode"`
	StorageStatus        string `json:"StorageStatus"`
	CID                  string `json:"CID"`
	Miner                string `json:"Miner"`
	StorageCost          string `json:"StorageCost"`
	StorageCostEstimated string `json:"StorageCostEstimated"`
	FilecoinDealExpiry   int64  `json:"FilecoinDealExpiry"`
	FFSToken             string `json:"FFSToken"`
	JobID                string `json:"JobID"`
}

// User is the type binding for HTTP Basic Authentication.
type User struct {
	Name       string `json:"Name"`
	TokenID    string `json:"TokenID"`
	Digest     string `json:"Digest"`
	AssetCount uint64 `json:"AssetCount"`
	CreatedAt  int64  `json:"CreatedAt"`
}

// Password   string `json:"Password"`
