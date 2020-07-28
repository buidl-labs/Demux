package dataservice

import (
	"database/sql"

	"github.com/buidl-labs/Demux/model"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

const dbFilePath = "./DemuxDB.sqlite3"

var sqldb *sql.DB

func InitDB() {
	database, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		log.Fatalln("Error in creating DB", err.Error())
	}
	sqldb = database

	statement, err := database.Prepare(`
		CREATE TABLE IF NOT EXISTS TranscodingDeal (
			TranscodingID   INT PRIMARY KEY,
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
			CID           TEXT PRIMARY KEY,
			Name          TEXT,
			Miner         TEXT,
			StorageCost   FLOAT,
			Expiry        UNSIGNED BIG INT,
			TranscodingID UNSIGNED BIG INT
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

func CreateTranscodingDeal(x model.TranscodingDeal) {}

func CreateStorageDeal(x model.StorageDeal) {}

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
