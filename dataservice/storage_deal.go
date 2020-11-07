package dataservice

import (
	"context"

	"github.com/buidl-labs/Demux/model"

	log "github.com/sirupsen/logrus"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StorageDealDatabase provides the storageDeal db operations.
type StorageDealDatabase interface {
	InsertStorageDeal(model.StorageDeal) error
	GetStorageDeal(string) (model.StorageDeal, error)
	GetStorageDealByCID(string) (model.StorageDeal, error)
	UpdateStorageDeal(string, uint32, string, string, string, int64) error
	GetPendingDeals() ([]model.StorageDeal, error)
}

type storageDealDatabase struct {
	db DatabaseHelper
}

// NewStorageDealDatabase returns an instance of StorageDealDatabase.
func NewStorageDealDatabase(db DatabaseHelper) StorageDealDatabase {
	return &storageDealDatabase{
		db: db,
	}
}

func (sd *storageDealDatabase) InsertStorageDeal(storageDeal model.StorageDeal) error {
	insertResult, err := sd.db.Collection("storageDeal").InsertOne(context.Background(), storageDeal)
	if err != nil {
		log.Error("Inserting a storageDeal:", err)
		return err
	}
	log.Info("Inserted a storageDeal: ", insertResult)
	return nil
}

func (sd *storageDealDatabase) GetStorageDeal(assetID string) (model.StorageDeal, error) {
	result := model.StorageDeal{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := sd.db.Collection("storageDeal").FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Warn("Getting storageDeal: ", err)
		return result, err
	}
	log.Info("Getting storageDeal: ", result.AssetID)
	return result, nil
}

func (sd *storageDealDatabase) GetStorageDealByCID(CID string) (model.StorageDeal, error) {
	result := model.StorageDeal{}
	filter := bson.D{primitive.E{Key: "cid", Value: CID}}
	err := sd.db.Collection("storageDeal").FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Warn("Getting storageDeal: ", err)
		return result, err
	}
	log.Info("Getting storageDeal: ", result.AssetID)
	return result, nil
}

func (sd *storageDealDatabase) UpdateStorageDeal(CID string, storageStatusCode uint32, storageStatus string, miner string, storageCost string, filecoinDealExpiry int64) error {
	filter := bson.M{"cid": CID}
	update := bson.M{"$set": bson.M{
		"storage_status_code":  storageStatusCode,
		"storage_status":       storageStatus,
		"miner":                miner,
		"storage_cost":         storageCost,
		"filecoin_deal_expiry": filecoinDealExpiry,
	}}
	result, err := sd.db.Collection("storageDeal").UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error("Updating storageDeal: ", err)
		return err
	}
	log.Info("Updating storageDeal: ", result)
	return nil
}

func (sd *storageDealDatabase) GetPendingDeals() ([]model.StorageDeal, error) {
	var results = []model.StorageDeal{}

	filter := bson.D{primitive.E{Key: "storage_status_code", Value: 0}}
	cur, err := sd.db.Collection("storageDeal").Find(context.Background(), filter)
	if err != nil {
		log.Error("Getting pending storage deals: ", err)
		return results, err
	}

	for cur.Next(context.Background()) {
		var result model.StorageDeal
		e := cur.Decode(&result)
		if e != nil {
			log.Error("Getting pending storage deals: ", e)
			return results, e
		}
		results = append(results, result)
	}
	if err := cur.Err(); err != nil {
		log.Error("Getting pending storage deals: ", err)
		return results, err
	}

	cur.Close(context.Background())
	log.Info("Getting pending storage deals: ", len(results))
	return results, nil
}
