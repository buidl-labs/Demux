package internal

// AssetStatusMap maps asset_status_code with the corresponding asset_status.
var AssetStatusMap = map[int32]string{
	-1: "asset created",
	0:  "video uploaded successfully",
	1:  "processing in livepeer",
	2:  "attempting to pin to ipfs",
	3:  "pinned to ipfs, attempting to store in filecoin",
	4:  "stored in filecoin",
}
