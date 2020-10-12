package dataservice

import (
	"database/sql"

	"github.com/buidl-labs/Demux/model"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

const dbFilePath = "./DemuxDB.sqlite3"

var sqldb *sql.DB

// InitDB initializes the Demux database.
func InitDB() {
	database, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		log.Fatalln("Error in creating DB", err)
	}
	sqldb = database

	statement, err := database.Prepare(`
		CREATE TABLE IF NOT EXISTS Asset (
			AssetID                  TEXT PRIMARY KEY,
			AssetReady               BOOLEAN,
			AssetStatusCode          INT,
			AssetStatus              TEXT,
			AssetError               BOOLEAN,
			StreamURL                TEXT,
			Thumbnail                TEXT,
			CreatedAt                INT
			CHECK (AssetStatusCode >= 0 AND AssetStatusCode <= 4)
		)
	`)

	if err != nil {
		log.Fatalln("Error in creating DB", err)
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatalln("Error in creating DB", err)
	}

	statement, err = database.Prepare(`
		CREATE TABLE IF NOT EXISTS Upload (
			AssetID         TEXT PRIMARY KEY,
			URL             TEXT,
			Status          BOOLEAN
		)
	`)

	if err != nil {
		log.Fatalln("Error in creating DB", err)
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatalln("Error in creating DB", err)
	}

	statement, err = database.Prepare(`
		CREATE TABLE IF NOT EXISTS TranscodingDeal (
			AssetID                  TEXT PRIMARY KEY,
			TranscodingCost          TEXT,
			TranscodingCostEstimated TEXT
		)
	`)
	if err != nil {
		log.Fatalln("Error in creating DB", err)
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatalln("Error in creating DB", err)
	}

	statement, err = database.Prepare(`
		CREATE TABLE IF NOT EXISTS StorageDeal (
			AssetID              TEXT PRIMARY KEY,
			StorageStatusCode    INT,
			StorageStatus        TEXT,
			CID                  TEXT,
			Miner                TEXT,
			StorageCost          TEXT,
			StorageCostEstimated TEXT,
			FilecoinDealExpiry   BIGINT,
			FFSToken             TEXT,
			JobID                TEXT
		)
	`)

	if err != nil {
		log.Fatalln("Error in creating DB", err)
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatalln("Error in creating DB", err)
	}

	statement, err = database.Prepare(`
		CREATE TABLE IF NOT EXISTS User (
			Name          TEXT,
			TokenID       TEXT,
			Digest        TEXT PRIMARY KEY,
			AssetCount    INT,
			CreatedAt     DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)

	if err != nil {
		log.Fatalln("Error in creating DB", err)
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatalln("Error in creating DB", err)
	}

	statement, err = database.Prepare(`
		CREATE TABLE IF NOT EXISTS SizeRatio (
			AssetID           TEXT PRIMARY KEY,
			SizeRatio         DOUBLE,
			VideoFileSize     INT,
			StreamFolderSize  INT
		)
	`)

	if err != nil {
		log.Fatalln("Error in creating DB", err)
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatalln("Error in creating DB", err)
	}

	statement, err = database.Prepare(`
		CREATE TABLE IF NOT EXISTS MeanSizeRatio (
			MeanSizeRatio  DOUBLE,
			RatioSum       DOUBLE,
			Count          INT
		)
	`)

	if err != nil {
		log.Fatalln("Error in creating DB", err)
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatalln("Error in creating DB", err)
	}

	log.Info("DB created successfully.")
}

// CreateAsset creates a new asset.
func CreateAsset(x model.Asset) {
	statement, err := sqldb.Prepare("INSERT INTO Asset (AssetID, AssetReady, AssetStatusCode, AssetStatus, AssetError, StreamURL, Thumbnail, CreatedAt) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Println("Error in inserting Asset", x.AssetID)
		log.Println(err)
	}
	_, err = statement.Exec(x.AssetID, x.AssetReady, x.AssetStatusCode, x.AssetStatus, x.AssetError, x.StreamURL, x.Thumbnail, x.CreatedAt)
	if err != nil {
		log.Println("Error in inserting Asset", x.AssetID)
		log.Println(err)
	}
}

// CreateUpload creates a new upload.
func CreateUpload(x model.Upload) {
	statement, err := sqldb.Prepare("INSERT INTO Upload (AssetID, URL, Status) VALUES (?, ?, ?)")
	if err != nil {
		log.Println("Error in inserting Upload", x.AssetID)
		log.Println(err)
	}
	_, err = statement.Exec(x.AssetID, x.URL, x.Status)
	if err != nil {
		log.Println("Error in inserting Upload", x.AssetID)
		log.Println(err)
	}
}

// UpdateAssetStatus updates the status of an asset.
func UpdateAssetStatus(assetID string, assetStatusCode uint32, assetStatus string, assetError bool) {
	statement, err := sqldb.Prepare("UPDATE Asset SET AssetStatusCode=?, AssetStatus=?, AssetError=? WHERE AssetID=?")
	if err != nil {
		log.Println("Error in updating asset", assetID)
		log.Println(err)
	}
	_, err = statement.Exec(assetStatusCode, assetStatus, assetError, assetID)
	if err != nil {
		log.Println("Error in updating asset", assetID)
		log.Println(err)
	}
}

// UpdateUploadStatus updates the status of an upload.
func UpdateUploadStatus(assetID string, status bool) {
	statement, err := sqldb.Prepare("UPDATE Upload SET Status=? WHERE AssetID=?")
	if err != nil {
		log.Println("Error in updating upload", assetID)
		log.Println(err)
	}
	_, err = statement.Exec(status, assetID)
	if err != nil {
		log.Println("Error in updating upload", assetID)
		log.Println(err)
	}
}

// CreateTranscodingDeal creates a new transcoding deal.
func CreateTranscodingDeal(x model.TranscodingDeal) {
	statement, err := sqldb.Prepare("INSERT INTO TranscodingDeal (AssetID, TranscodingCost, TranscodingCostEstimated) VALUES (?, ?, ?)")
	if err != nil {
		log.Println("Error in inserting TranscodingDeal", x.AssetID)
		log.Println(err)
	}
	_, err = statement.Exec(x.AssetID, x.TranscodingCost, x.TranscodingCostEstimated)
	if err != nil {
		log.Println("Error in inserting TranscodingDeal", x.AssetID)
		log.Println(err)
	}
}

// CreateStorageDeal creates a new storage deal.
func CreateStorageDeal(x model.StorageDeal) {
	statement, err := sqldb.Prepare("INSERT INTO StorageDeal (AssetID, StorageStatusCode, StorageStatus, CID, Miner, StorageCost, StorageCostEstimated, FilecoinDealExpiry, FFSToken, JobID) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Println("Error in inserting StorageDeal", x.AssetID)
		log.Println(err)
	}
	_, err = statement.Exec(x.AssetID, x.StorageStatusCode, x.StorageStatus, x.CID, x.Miner, x.StorageCost, x.StorageCostEstimated, x.FilecoinDealExpiry, x.FFSToken, x.JobID)
	if err != nil {
		log.Println("Error in inserting StorageDeal", x.AssetID)
		log.Println(err)
	}
}

// IfAssetExists checks whether a given asset exists in the database.
func IfAssetExists(assetID string) bool {
	count := 0
	rows, err := sqldb.Query("SELECT * FROM Asset WHERE AssetID=?", assetID)
	if err != nil {
		log.Println("Error in checking existence of asset", assetID)
		log.Println(err)
	}
	for rows.Next() {
		count++
	}
	if count == 0 {
		return false
	}
	return true
}

// IfUploadExists checks whether a given upload exists in the database.
func IfUploadExists(assetID string) bool {
	count := 0
	rows, err := sqldb.Query("SELECT * FROM Upload WHERE AssetID=?", assetID)
	if err != nil {
		log.Println("Error in checking existence of upload", assetID)
		log.Println(err)
	}
	for rows.Next() {
		count++
	}
	if count == 0 {
		return false
	}
	return true
}

// GetAsset returns an asset.
func GetAsset(assetID string) model.Asset {
	rows, err := sqldb.Query("SELECT * FROM Asset WHERE AssetID=?", assetID)
	if err != nil {
		log.Println("Error in getting asset", assetID)
		log.Println(err)
	}
	var data []model.Asset
	x := model.Asset{}
	for rows.Next() {
		rows.Scan(&x.AssetID, &x.AssetReady, &x.AssetStatusCode, &x.AssetStatus, &x.AssetError, &x.StreamURL, &x.Thumbnail, &x.CreatedAt)
		data = append(data, x)
	}
	return x
}

// GetUpload returns an upload.
func GetUpload(assetID string) model.Upload {
	rows, err := sqldb.Query("SELECT * FROM Upload WHERE AssetID=?", assetID)
	if err != nil {
		log.Println("Error in getting upload", assetID)
		log.Println(err)
	}
	var data []model.Upload
	x := model.Upload{}
	for rows.Next() {
		rows.Scan(&x.AssetID, &x.URL, &x.Status)
		data = append(data, x)
	}
	return x
}

// GetTranscodingDeal returns a transcoding deal.
func GetTranscodingDeal(assetID string) model.TranscodingDeal {
	rows, err := sqldb.Query("SELECT * FROM TranscodingDeal WHERE AssetID=?", assetID)
	if err != nil {
		log.Println("Error in getting TranscodingDeal", assetID)
		log.Println(err)
	}
	data := []model.TranscodingDeal{}
	x := model.TranscodingDeal{}
	for rows.Next() {
		rows.Scan(&x.AssetID, &x.TranscodingCost, &x.TranscodingCostEstimated)
		data = append(data, x)
	}
	return x
}

// GetStorageDeal returns a storage deal.
func GetStorageDeal(assetID string) model.StorageDeal {
	rows, err := sqldb.Query("SELECT * FROM StorageDeal WHERE AssetID=?", assetID)
	if err != nil {
		log.Println("Error in getting StorageDeal", assetID)
		log.Println(err)
	}
	data := []model.StorageDeal{}
	x := model.StorageDeal{}
	for rows.Next() {
		rows.Scan(&x.AssetID, &x.StorageStatusCode, &x.StorageStatus, &x.CID, &x.Miner, &x.StorageCost, &x.StorageCostEstimated, &x.FilecoinDealExpiry, &x.FFSToken, &x.JobID)
		data = append(data, x)
	}
	return x
}

// GetPendingDeals returns the pending storage deals.
func GetPendingDeals() []model.StorageDeal {
	rows, err := sqldb.Query("SELECT * FROM StorageDeal WHERE StorageStatusCode=?", 0)
	if err != nil {
		log.Println("Error in getting asset")
		log.Println(err)
	}
	data := []model.StorageDeal{}
	x := model.StorageDeal{}
	for rows.Next() {
		rows.Scan(&x.AssetID, &x.StorageStatusCode, &x.StorageStatus, &x.CID, &x.Miner, &x.StorageCost, &x.StorageCostEstimated, &x.FilecoinDealExpiry, &x.FFSToken, &x.JobID)
		data = append(data, x)
	}
	return data
}

// UpdateStorageDeal updates a storage deal.
func UpdateStorageDeal(CID string, storageStatusCode uint32, storageStatus string, miner string, storageCost string, filecoinDealExpiry int64) {
	statement, err := sqldb.Prepare("UPDATE StorageDeal SET StorageStatusCode=?, StorageStatus=?, Miner=?, StorageCost=?, FilecoinDealExpiry=? WHERE CID=?")
	if err != nil {
		log.Println("Error in updating storage deal having CID", CID)
		log.Println(err)
	}
	_, err = statement.Exec(storageStatusCode, storageStatus, miner, storageCost, filecoinDealExpiry, CID)
	if err != nil {
		log.Println("Error in updating storage deal having CID", CID)
		log.Println(err)
	}
}

// UpdateStreamURL updates the StreamURL of an asset.
func UpdateStreamURL(assetID string, streamURL string) {
	statement, err := sqldb.Prepare("UPDATE Asset SET StreamURL=? WHERE AssetID=?")
	if err != nil {
		log.Println("Error in updating streamURL for asset", assetID)
		log.Println(err)
	}
	_, err = statement.Exec(streamURL, assetID)
	if err != nil {
		log.Println("Error in updating streamURL for asset", assetID)
		log.Println(err)
	}
}

// UpdateThumbnail updates the thumbnail of an asset.
func UpdateThumbnail(assetID string, thumbnail string) {
	statement, err := sqldb.Prepare("UPDATE Asset SET Thumbnail=? WHERE AssetID=?")
	if err != nil {
		log.Println("Error in updating thumbnail for asset", assetID)
		log.Println(err)
	}
	_, err = statement.Exec(thumbnail, assetID)
	if err != nil {
		log.Println("Error in updating thumbnail for asset", assetID)
		log.Println(err)
	}
}

// CreateUser creates a new user.
func CreateUser(x model.User) {
	statement, err := sqldb.Prepare("INSERT INTO User (Name, TokenID, Digest, AssetCount, CreatedAt) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		log.Println("Error in inserting User", x.Name)
		log.Println(err)
	}
	_, err = statement.Exec(x.Name, x.TokenID, x.Digest, x.AssetCount, x.CreatedAt)
	if err != nil {
		log.Println("Error in inserting User", x.Name)
		log.Println(err)
	}
}

// IfUserExists checks whether a user having a given digest exists in the database.
func IfUserExists(digest string) bool {
	count := 0
	rows, err := sqldb.Query("SELECT * FROM User WHERE Digest=?", digest)
	if err != nil {
		log.Println("Error in checking existence of user", digest)
		log.Println(err)
	}
	for rows.Next() {
		count++
	}
	if count == 0 {
		return false
	}
	return true
}

// IncrementUserAssetCount increments a user's AssetCount by 1.
func IncrementUserAssetCount(digest string) {
	statement, err := sqldb.Prepare("UPDATE User SET AssetCount = AssetCount + 1 WHERE Digest=?")
	if err != nil {
		log.Println("Error in updating assetCount for user", digest)
		log.Println(err)
	}
	_, err = statement.Exec(digest)
	if err != nil {
		log.Println("Error in updating assetCount for user", digest)
		log.Println(err)
	}
}

// UpdateAssetReady updates the "ready" state of an asset.
func UpdateAssetReady(assetID string, assetReady bool) {
	statement, err := sqldb.Prepare("UPDATE Asset SET AssetReady=? WHERE AssetID=?")
	if err != nil {
		log.Println("Error in updating asset", assetID)
		log.Println(err)
	}
	_, err = statement.Exec(assetReady, assetID)
	if err != nil {
		log.Println("Error in updating asset", assetID)
		log.Println(err)
	}
}

// AddSizeRatio adds a new SizeRatio.
func AddSizeRatio(x model.SizeRatio) {
	statement, err := sqldb.Prepare("INSERT INTO SizeRatio (AssetID, SizeRatio, VideoFileSize, StreamFolderSize) VALUES (?, ?, ?, ?)")
	if err != nil {
		log.Println("Error in inserting SizeRatio for asset", x.AssetID)
		log.Println(err)
	}
	_, err = statement.Exec(x.AssetID, x.SizeRatio, x.VideoFileSize, x.StreamFolderSize)
	if err != nil {
		log.Println("Error in inserting SizeRatio for asset", x.AssetID)
		log.Println(err)
	}
}

// UpdateMeanSizeRatio updates the mean size ratio .
func UpdateMeanSizeRatio(ratio float64, ratioSum float64, count uint64) {
	statement, err := sqldb.Prepare("UPDATE MeanSizeRatio SET MeanSizeRatio=?, RatioSum=?, Count=?")
	if err != nil {
		log.Println("Error in updating MeanSizeRatio", ratio)
		log.Println(err)
	}
	_, err = statement.Exec(ratio, ratioSum, count)
	if err != nil {
		log.Println("Error in updating MeanSizeRatio", ratio)
		log.Println(err)
	}
}

// GetMeanSizeRatio returns the current mean size ratio.
func GetMeanSizeRatio() model.MeanSizeRatio {
	rows, err := sqldb.Query("SELECT * FROM MeanSizeRatio")
	if err != nil {
		log.Println("Error in getting MeanSizeRatio")
		log.Println(err)
	}
	data := []model.MeanSizeRatio{}
	x := model.MeanSizeRatio{}
	for rows.Next() {
		rows.Scan(&x.MeanSizeRatio, &x.RatioSum, &x.Count)
		data = append(data, x)
	}
	return x
}
