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
		log.Fatalln("a1Error in creating DB", err.Error())
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatalln("a2Error in creating DB", err.Error())
	}

	statement, err = database.Prepare(`
		CREATE TABLE IF NOT EXISTS StorageDeal (
			CID           TEXT PRIMARY KEY,
			RootCID       TEXT,
			Name          TEXT,
			AssetID       TEXT,
			Miner         TEXT,
			StorageCost   FLOAT,
			Expiry        UNSIGNED BIG INT,
			TranscodingID TEXT
		)
	`)

	if err != nil {
		log.Fatalln("s1Error in creating DB", err.Error())
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatalln("s2Error in creating DB", err.Error())
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
			CHECK (AssetStatus >= 0 AND AssetStatus <= 3)
		)
	`)

	if err != nil {
		log.Fatalln("h1Error in creating DB", err.Error())
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatalln("h2Error in creating DB", err.Error())
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
	statement, err := sqldb.Prepare("INSERT INTO StorageDeal (CID, Name, AssetID, Miner, StorageCost, Expiry, TranscodingID) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Errorln("Error in inserting StorageDeal", x.CID)
		log.Errorln(err.Error())
	}
	_, err = statement.Exec(x.CID, x.Name, x.AssetID, x.Miner, x.StorageCost, x.Expiry, x.TranscodingID)
	if err != nil {
		log.Errorln("Error in inserting StorageDeal", x.CID)
		log.Errorln(err.Error())
	}
}

// GetCIDsForAsset returns the final stream CIDs for the given asset.
func GetCIDsForAsset(assetID string) (string, string) {
	rows, err := sqldb.Query("SELECT * FROM StorageDeal WHERE AssetID=?", assetID)
	if err != nil {
		log.Errorln("Error in getting CIDs for asset", assetID)
		log.Errorln(err.Error())
	}
	data := []model.StorageDeal{}
	x := model.StorageDeal{}
	for rows.Next() {
		rows.Scan(&x.CID, &x.RootCID, &x.Name, &x.AssetID, &x.Miner, &x.StorageCost, &x.Expiry, &x.TranscodingID)
		data = append(data, x)
	}
	return data[0].CID, data[0].RootCID
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
func UpdateStorageDeal(CID string, rootCID string, storageCost float64, expiry uint32) {
	statement, err := sqldb.Prepare("UPDATE StorageDeal SET RootCID=?, StorageCost=?, Expiry=? WHERE CID=?")
	if err != nil {
		log.Errorln("Error in updating storage deal", CID)
		log.Errorln(err.Error())
	}
	_, err = statement.Exec(rootCID, storageCost, expiry, CID)
	if err != nil {
		log.Errorln("Error in updating storage deal", CID)
		log.Errorln(err.Error())
	}
}
