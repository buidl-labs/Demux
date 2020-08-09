package model

// Asset is the entity (video) which is uploaded by the user.
type Asset struct {
	AssetID     string `json:"AssetID"`
	AssetName   string `json:"AssetName"`
	AssetStatus int    `json:"AssetStatus"`
}

// TranscodingDeal is the type binding for a transcoding deal in the livepeer network.
type TranscodingDeal struct {
	TranscodingID   string  `json:"TranscodingID"`
	TranscodingCost float64 `json:"TranscodingCost"`
	Directory       string  `json:"Directory"`
	StorageStatus   bool    `json:"StorageStatus"`
}

// StorageDeal is the type binding for a storage deal in the filecoin network.
type StorageDeal struct {
	DealID        string  `json:"DealID"`
	CID1080p      string  `json:"CID1080p"`
	CID720p       string  `json:"CID720p"`
	CID360p       string  `json:"CID360p"`
	AssetID       string  `json:"AssetID"`
	Miner         string  `json:"Miner"`
	StorageCost   float64 `json:"Cost"`
	Expiry        uint32  `json:"Expiry"`
	TranscodingID string  `json:"TranscodingID"`
}
