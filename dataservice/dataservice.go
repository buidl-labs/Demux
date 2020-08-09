package dataservice

import (
	"database/sql"
	"fmt"

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
			TranscodingCost FLOAT,
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
			DealID        TEXT PRIMARY KEY,
			CID1080p      TEXT,
			CID720p       TEXT,
			CID360p       TEXT,
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
			AssetID       TEXT PRIMARY KEY,
			AssetName     TEXT,
			AssetStatus   INT,
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
	statement, err := sqldb.Prepare("INSERT INTO StorageDeal (DealID, CID1080p, CID720p, CID360p, AssetID, Miner, StorageCost, Expiry, TranscodingID) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Errorln("Error in inserting StorageDeal", x.DealID)
		log.Errorln(err.Error())
	}
	_, err = statement.Exec(x.DealID, x.CID1080p, x.CID720p, x.CID360p, x.AssetID, x.Miner, x.StorageCost, x.Expiry, x.TranscodingID)
	if err != nil {
		log.Errorln("Error in inserting StorageDeal", x.DealID)
		log.Errorln(err.Error())
	}
}

// GetDealForAsset returns the storage deal for the given asset.
func GetDealForAsset(assetID string) (model.StorageDeal, error) {
	rows, err := sqldb.Query("SELECT * FROM StorageDeal WHERE AssetID=?", assetID)
	if err != nil {
		log.Errorln("Error in getting Deal for asset", assetID)
		log.Errorln(err.Error())
	}
	data := []model.StorageDeal{}
	x := model.StorageDeal{}
	dummydata := model.StorageDeal{
		DealID: "0",
	}
	for rows.Next() {
		rows.Scan(&x.DealID, &x.CID1080p, &x.CID720p, &x.CID360p, &x.AssetID, &x.Miner, &x.StorageCost, &x.Expiry, &x.TranscodingID)
		data = append(data, x)
	}
	fmt.Println("getdeal", data)
	if len(data) == 0 {
		return dummydata, fmt.Errorf("No storage deal made yet")
	}
	return data[0], nil
}

// CreateAsset creates a new asset.
func CreateAsset(x model.Asset) {
	statement, err := sqldb.Prepare("INSERT INTO Asset (AssetID, AssetName, AssetStatus) VALUES (?, ?, ?)")
	if err != nil {
		log.Errorln("Error in inserting Asset", x.AssetID)
		log.Errorln(err.Error())
	}
	_, err = statement.Exec(x.AssetID, x.AssetName, x.AssetStatus)
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
		rows.Scan(&x.AssetID, &x.AssetName, &x.AssetStatus)
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
