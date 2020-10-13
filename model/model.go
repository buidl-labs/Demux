package model

// Asset is the entity (video) which is uploaded by the user.
type Asset struct {
	AssetID         string `bson:"_id" json:"_id"`
	AssetReady      bool   `bson:"asset_ready" json:"asset_ready"`
	AssetStatusCode uint32 `bson:"asset_status_code" json:"asset_status_code"`
	AssetStatus     string `bson:"asset_status" json:"asset_status"`
	AssetError      bool   `bson:"asset_error" json:"asset_error"`
	StreamURL       string `bson:"stream_url" json:"stream_url"`
	Thumbnail       string `bson:"thumbnail" json:"thumbnail"`
	CreatedAt       int64  `bson:"created_at" json:"created_at"`
}

// Upload is the entity (video) which is uploaded by the client.
type Upload struct {
	AssetID string `bson:"_id" json:"_id"`
	URL     string `bson:"url" json:"url"`
	Status  bool   `bson:"status" json:"status"`
}

// TranscodingDeal is the type binding for a transcoding deal in the livepeer network.
type TranscodingDeal struct {
	AssetID                  string `bson:"_id" json:"_id"`
	TranscodingCost          string `bson:"transcoding_cost" json:"transcoding_cost"`
	TranscodingCostEstimated string `bson:"transcoding_cost_estimated" json:"transcoding_cost_estimated"`
}

// StorageDeal is the type binding for a storage deal in the IPFS/filecoin network.
type StorageDeal struct {
	AssetID              string `bson:"_id" json:"_id"`
	StorageStatusCode    uint32 `bson:"storage_status_code" json:"storage_status_code"`
	StorageStatus        string `bson:"storage_status" json:"storage_status"`
	CID                  string `bson:"cid" json:"cid"`
	Miner                string `bson:"miner" json:"miner"`
	StorageCost          string `bson:"storage_cost" json:"storage_cost"`
	StorageCostEstimated string `bson:"storage_cost_estimated" json:"storage_cost_estimated"`
	FilecoinDealExpiry   int64  `bson:"filecoin_deal_expiry" json:"filecoin_deal_expiry"`
	FFSToken             string `bson:"ffs_token" json:"ffs_token"`
	JobID                string `bson:"job_id" json:"job_id"`
}

// User is the type binding for HTTP Basic Authentication.
type User struct {
	Name       string `bson:"name" json:"name"`
	TokenID    string `bson:"token_id" json:"token_id"`
	Digest     string `bson:"digest" json:"digest"`
	AssetCount uint64 `bson:"asset_count" json:"asset_count"`
	CreatedAt  int64  `bson:"created_at" json:"created_at"`
}

// SizeRatio store the ratio StreamFolderSize/VideoFileSize.
type SizeRatio struct {
	AssetID          string  `bson:"_id" json:"_id"`
	SizeRatio        float64 `bson:"size_ratio" json:"size_ratio"`
	VideoFileSize    uint64  `bson:"video_file_fize" json:"video_file_fize"`
	StreamFolderSize uint64  `bson:"stream_folder_size" json:"stream_folder_size"`
}

// MeanSizeRatio stores the mean SizeRatio of all assets.
type MeanSizeRatio struct {
	ID            int64   `bson:"_id" json:"_id"`
	MeanSizeRatio float64 `bson:"ratio" json:"ratio"`
	RatioSum      float64 `bson:"ratio_sum" json:"ratio_sum"`
	Count         uint64  `bson:"count" json:"count"`
}
