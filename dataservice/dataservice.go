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
		log.Fatalln("Error in creating DB", err.Error())
	}
	sqldb = database

	statement, err := database.Prepare(`
		CREATE TABLE IF NOT EXISTS TranscodingDeal (
			TranscodingID   TEXT PRIMARY KEY,
			TranscodingCost TEXT,
			Directory       TEXT,
			StorageStatus   BOOLEAN
		)
	`)
	if err != nil {
		log.Fatalln("Error in creating DB", err.Error())
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatalln("Error in creating DB", err.Error())
	}

	statement, err = database.Prepare(`
		CREATE TABLE IF NOT EXISTS StorageDeal (
			CID           TEXT PRIMARY KEY,
			Name          TEXT,
			AssetID       TEXT,
			Miner         TEXT,
			StorageCost   FLOAT,
			Expiry        UNSIGNED BIG INT,
			TranscodingID TEXT,
			Token         TEXT,
			JID           TEXT,
			Status        INT
		)
	`)

	if err != nil {
		log.Fatalln("Error in creating DB", err.Error())
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatalln("Error in creating DB", err.Error())
	}

	statement, err = database.Prepare(`
		CREATE TABLE IF NOT EXISTS Asset (
			AssetID         TEXT PRIMARY KEY,
			AssetName       TEXT,
			AssetStatus     INT,
			TranscodingCost TEXT,
			Miner           TEXT,
			StorageCost     FLOAT,
			Expiry          INT,
			Error           TEXT,
			HttpStatusCode  INT
			CHECK (AssetStatus >= 0 AND AssetStatus <= 4)
		)
	`)

	if err != nil {
		log.Fatalln("Error in creating DB", err.Error())
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatalln("Error in creating DB", err.Error())
	}

	log.Info("DB created successfully.")
}

// CreateTranscodingDeal creates a new transcoding deal.
func CreateTranscodingDeal(x model.TranscodingDeal) {
	statement, err := sqldb.Prepare("INSERT INTO TranscodingDeal (TranscodingID, TranscodingCost, Directory, StorageStatus) VALUES (?, ?, ?, ?)")
	if err != nil {
		log.Errorln("Error in inserting TranscodingDeal", x.TranscodingID)
		log.Errorln(err.Error())
	}
	_, err = statement.Exec(x.TranscodingID, x.TranscodingCost, x.Directory, x.StorageStatus)
	if err != nil {
		log.Errorln("Error in inserting TranscodingDeal", x.TranscodingID)
		log.Errorln(err.Error())
	}
}

// CreateStorageDeal creates a new storage deal.
func CreateStorageDeal(x model.StorageDeal) {
	statement, err := sqldb.Prepare("INSERT INTO StorageDeal (CID, Name, AssetID, Miner, StorageCost, Expiry, TranscodingID, Token, JID, Status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Errorln("Error in inserting StorageDeal", x.CID)
		log.Errorln(err.Error())
	}
	_, err = statement.Exec(x.CID, x.Name, x.AssetID, x.Miner, x.StorageCost, x.Expiry, x.TranscodingID, x.Token, x.JID, x.Status)
	if err != nil {
		log.Errorln("Error in inserting StorageDeal", x.CID)
		log.Errorln(err.Error())
	}
}

// GetCIDForAsset returns the final stream CIDs for the given asset.
func GetCIDForAsset(assetID string) string {
	rows, err := sqldb.Query("SELECT * FROM StorageDeal WHERE AssetID=?", assetID)
	if err != nil {
		log.Errorln("Error in getting CIDs for asset", assetID)
		log.Errorln(err.Error())
	}
	data := []model.StorageDeal{}
	x := model.StorageDeal{}
	for rows.Next() {
		rows.Scan(&x.CID, &x.Name, &x.AssetID, &x.Miner, &x.StorageCost, &x.Expiry, &x.TranscodingID, &x.Token, &x.JID, &x.Status)
		data = append(data, x)
	}
	return data[0].CID
}

// GetAsset returns an asset.
func GetAsset(assetID string) model.Asset {
	rows, err := sqldb.Query("SELECT * FROM Asset WHERE AssetID=?", assetID)
	if err != nil {
		log.Errorln("Error in getting asset", assetID)
		log.Errorln(err.Error())
	}
	var data []model.Asset
	x := model.Asset{}
	for rows.Next() {
		rows.Scan(&x.AssetID, &x.AssetName, &x.AssetStatus, &x.TranscodingCost, &x.Miner, &x.StorageCost, &x.Expiry, &x.Error, &x.HttpStatusCode)
		data = append(data, x)
	}
	return data[0]
}

// GetAssetError returns an asset error.
func GetAssetError(assetID string) string {
	rows, err := sqldb.Query("SELECT Error FROM Asset WHERE AssetID=?", assetID)
	if err != nil {
		log.Errorln("Error in getting asset error", assetID)
		log.Errorln(err.Error())
	}
	var data []string
	var x string
	for rows.Next() {
		rows.Scan(&x)
		data = append(data, x)
	}
	return data[0]
}

// CreateAsset creates a new asset.
func CreateAsset(x model.Asset) {
	statement, err := sqldb.Prepare("INSERT INTO Asset (AssetID, AssetName, AssetStatus, Error) VALUES (?, ?, ?, ?)")
	if err != nil {
		log.Errorln("Error in inserting Asset", x.AssetID)
		log.Errorln(err.Error())
	}
	_, err = statement.Exec(x.AssetID, x.AssetName, x.AssetStatus, x.Error)
	if err != nil {
		log.Errorln("Error in inserting Asset", x.AssetID)
		log.Errorln(err.Error())
	}
}

// IfAssetExists checks whether a given asset exists in the database.
func IfAssetExists(assetID string) bool {
	count := 0
	rows, err := sqldb.Query("SELECT * FROM Asset WHERE AssetID=?", assetID)
	if err != nil {
		log.Errorln("Error in checking existence of asset", assetID)
		log.Errorln(err.Error())
	}
	for rows.Next() {
		count++
	}
	if count == 0 {
		return false
	}
	return true
}

// GetAssetStatusIfExists returns an asset if it exists in the database.
func GetAssetStatusIfExists(assetID string) int {
	rows, err := sqldb.Query("SELECT * FROM Asset WHERE AssetID=?", assetID)
	if err != nil {
		log.Errorln("Error in getting asset", assetID)
		log.Errorln(err.Error())
	}
	data := []model.Asset{}
	x := model.Asset{}
	for rows.Next() {
		rows.Scan(&x.AssetID, &x.AssetName, &x.AssetStatus, &x.TranscodingCost, &x.Miner, &x.StorageCost, &x.Expiry, &x.Error, &x.HttpStatusCode)
		data = append(data, x)
	}
	return data[0].AssetStatus
}

// UpdateAssetStatus updates the status of an asset.
func UpdateAssetStatus(assetID string, assetStatus int) {
	statement, err := sqldb.Prepare("UPDATE Asset SET AssetStatus=? WHERE AssetID=?")
	if err != nil {
		log.Errorln("Error in updating asset", assetID)
		log.Errorln(err.Error())
	}
	_, err = statement.Exec(assetStatus, assetID)
	if err != nil {
		log.Errorln("Error in updating asset", assetID)
		log.Errorln(err.Error())
	}
}

// UpdateAsset updates an asset.
func UpdateAsset(assetID string, transcodingCost string, miner string, storageCost float64, expiry uint32) {
	statement, err := sqldb.Prepare("UPDATE Asset SET TranscodingCost=?, Miner=?, StorageCost=?, Expiry=? WHERE AssetID=?")
	if err != nil {
		log.Errorln("Error in updating asset", assetID)
		log.Errorln(err.Error())
	}
	_, err = statement.Exec(transcodingCost, miner, storageCost, expiry, assetID)
	if err != nil {
		log.Errorln("Error in updating asset", assetID)
		log.Errorln(err.Error())
	}
}

// SetAssetError updates the error message for the uploaded asset.
func SetAssetError(assetID string, errorStr string, httpStatusCode int) {
	statement, err := sqldb.Prepare("UPDATE Asset SET Error=?, HttpStatusCode=? WHERE AssetID=?")
	if err != nil {
		log.Errorln("Error in updating asset error", assetID)
		log.Errorln(err.Error())
	}
	_, err = statement.Exec(errorStr, httpStatusCode, assetID)
	if err != nil {
		log.Errorln("Error in updating asset error", assetID)
		log.Errorln(err.Error())
	}
}

// UpdateStorageDeal updates a storage deal.
func UpdateStorageDeal(CID string, storageCost float64, expiry uint32, miner string) {
	statement, err := sqldb.Prepare("UPDATE StorageDeal SET StorageCost=?, Expiry=?, Miner=? WHERE CID=?")
	if err != nil {
		log.Errorln("Error in updating storage deal", CID)
		log.Errorln(err.Error())
	}
	_, err = statement.Exec(storageCost, expiry, miner, CID)
	if err != nil {
		log.Errorln("Error in updating storage deal", CID)
		log.Errorln(err.Error())
	}
}

// GetPendingDeals returns the pending storage deals.
func GetPendingDeals() []model.StorageDeal {
	rows, err := sqldb.Query("SELECT * FROM StorageDeal WHERE Status=?", 0)
	if err != nil {
		log.Errorln("Error in getting asset")
		log.Errorln(err.Error())
	}
	data := []model.StorageDeal{}
	x := model.StorageDeal{}
	for rows.Next() {
		rows.Scan(&x.CID, &x.Name, &x.AssetID, &x.Miner, &x.StorageCost, &x.Expiry, &x.TranscodingID, &x.Token, &x.JID, &x.Status)
		data = append(data, x)
	}
	return data
}

// UpdateStorageDealStatus updates a storage deal status.
func UpdateStorageDealStatus(CID string, status uint32) {
	statement, err := sqldb.Prepare("UPDATE StorageDeal SET Status=? WHERE CID=?")
	if err != nil {
		log.Errorln("Error in updating storage deal", CID)
		log.Errorln(err.Error())
	}
	_, err = statement.Exec(status, CID)
	if err != nil {
		log.Errorln("Error in updating storage deal", CID)
		log.Errorln(err.Error())
	}
}

// GetStorageDealStatus returns the status of a storage deal
// having a particular assetID.
func GetStorageDealStatus(assetID string) uint32 {
	rows, err := sqldb.Query("SELECT * FROM StorageDeal WHERE AssetID=?", assetID)
	if err != nil {
		log.Errorln("Error in getting StorageDeal", assetID)
		log.Errorln(err.Error())
	}
	data := []model.StorageDeal{}
	x := model.StorageDeal{}
	for rows.Next() {
		rows.Scan(&x.CID, &x.Name, &x.AssetID, &x.Miner, &x.StorageCost, &x.Expiry, &x.TranscodingID, &x.Token, &x.JID, &x.Status)
		data = append(data, x)
	}
	return data[0].Status
}

// GetStorageDeal returns a storage deal
// having a particular assetID.
func GetStorageDeal(assetID string) model.StorageDeal {
	rows, err := sqldb.Query("SELECT * FROM StorageDeal WHERE AssetID=?", assetID)
	if err != nil {
		log.Errorln("Error in getting StorageDeal", assetID)
		log.Errorln(err.Error())
	}
	data := []model.StorageDeal{}
	x := model.StorageDeal{}
	for rows.Next() {
		rows.Scan(&x.CID, &x.Name, &x.AssetID, &x.Miner, &x.StorageCost, &x.Expiry, &x.TranscodingID, &x.Token, &x.JID, &x.Status)
		data = append(data, x)
	}
	return data[0]
}
