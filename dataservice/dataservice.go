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
			AssetStatusCode          INT,
			AssetStatus              TEXT,
			AssetError               BOOLEAN,
			StreamURL                TEXT,
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
			CreatedAt     INT
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
	statement, err := sqldb.Prepare("INSERT INTO Asset (AssetID, AssetStatusCode, AssetStatus, AssetError, StreamURL, CreatedAt) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Println("Error in inserting Asset", x.AssetID)
		log.Println(err)
	}
	_, err = statement.Exec(x.AssetID, x.AssetStatusCode, x.AssetStatus, x.AssetError, x.StreamURL, x.CreatedAt)
	if err != nil {
		log.Println("Error in inserting Asset", x.AssetID)
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
		rows.Scan(&x.AssetID, &x.AssetStatusCode, &x.AssetStatus, &x.AssetError, &x.StreamURL, &x.CreatedAt)
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
	rows, err := sqldb.Query("SELECT * FROM StorageDeal WHERE StorageStatus=?", 0)
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

// ****************************************
// ****************************************
// ****************************************
// ****************************************
// ****************************************
// ****************************************

/*

// GetCIDForAsset returns the final stream CIDs for the given asset.
func GetCIDForAsset(assetID string) string {
	rows, err := sqldb.Query("SELECT * FROM StorageDeal WHERE AssetID=?", assetID)
	if err != nil {
		log.Println("Error in getting CIDs for asset", assetID)
		log.Println(err)
	}
	data := []model.StorageDeal{}
	x := model.StorageDeal{}
	for rows.Next() {
		rows.Scan(&x.CID, &x.Name, &x.AssetID, &x.Miner, &x.StorageCost, &x.Expiry, &x.TranscodingID, &x.Token, &x.JID, &x.Status)
		data = append(data, x)
	}
	return data[0].CID
}

// GetAssetError returns an asset error.
func GetAssetError(assetID string) string {
	rows, err := sqldb.Query("SELECT Error FROM Asset WHERE AssetID=?", assetID)
	if err != nil {
		log.Println("Error in getting asset error", assetID)
		log.Println(err)
	}
	var data []string
	var x string
	for rows.Next() {
		rows.Scan(&x)
		data = append(data, x)
	}
	return data[0]
}

// GetAssetStatusIfExists returns an asset if it exists in the database.
func GetAssetStatusIfExists(assetID string) int {
	rows, err := sqldb.Query("SELECT * FROM Asset WHERE AssetID=?", assetID)
	if err != nil {
		log.Println("Error in getting asset", assetID)
		log.Println(err)
	}
	data := []model.Asset{}
	x := model.Asset{}
	for rows.Next() {
		rows.Scan(&x.AssetID, &x.AssetName, &x.AssetStatus, &x.TranscodingCost, &x.Miner, &x.StorageCost, &x.Expiry, &x.Error, &x.StreamURL)
		data = append(data, x)
	}
	return data[0].AssetStatus
}

// UpdateAsset updates an asset.
func UpdateAsset(assetID string, transcodingCost string, miner string, storageCost float64, expiry uint32, streamURL string) {
	statement, err := sqldb.Prepare("UPDATE Asset SET TranscodingCost=?, Miner=?, StorageCost=?, Expiry=?, StreamURL=? WHERE AssetID=?")
	if err != nil {
		log.Println("Error in updating asset", assetID)
		log.Println(err)
	}
	_, err = statement.Exec(transcodingCost, miner, storageCost, expiry, streamURL, assetID)
	if err != nil {
		log.Println("Error in updating asset", assetID)
		log.Println(err)
	}
}

// SetAssetError updates the error message for the uploaded asset.
func SetAssetError(assetID string, errorStr string, httpStatusCode int) {
	statement, err := sqldb.Prepare("UPDATE Asset SET Error=?, HttpStatusCode=? WHERE AssetID=?")
	if err != nil {
		log.Println("Error in updating asset error", assetID)
		log.Println(err)
	}
	_, err = statement.Exec(errorStr, httpStatusCode, assetID)
	if err != nil {
		log.Println("Error in updating asset error", assetID)
		log.Println(err)
	}
}


// UpdateStorageDealStatus updates a storage deal status.
func UpdateStorageDealStatus(CID string, status uint32) {
	statement, err := sqldb.Prepare("UPDATE StorageDeal SET Status=? WHERE CID=?")
	if err != nil {
		log.Println("Error in updating storage deal", CID)
		log.Println(err)
	}
	_, err = statement.Exec(status, CID)
	if err != nil {
		log.Println("Error in updating storage deal", CID)
		log.Println(err)
	}
}

// GetStorageDealStatus returns the status of a storage deal
// having a particular assetID.
func GetStorageDealStatus(assetID string) uint32 {
	rows, err := sqldb.Query("SELECT * FROM StorageDeal WHERE AssetID=?", assetID)
	if err != nil {
		log.Println("Error in getting StorageDeal", assetID)
		log.Println(err)
	}
	data := []model.StorageDeal{}
	x := model.StorageDeal{}
	for rows.Next() {
		rows.Scan(&x.CID, &x.Name, &x.AssetID, &x.Miner, &x.StorageCost, &x.Expiry, &x.TranscodingID, &x.Token, &x.JID, &x.Status)
		data = append(data, x)
	}
	return data[0].Status
}

*/
