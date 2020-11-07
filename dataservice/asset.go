package dataservice

import (
	"context"

	"github.com/buidl-labs/Demux/model"

	log "github.com/sirupsen/logrus"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AssetDatabase provides the asset db operations.
type AssetDatabase interface {
	InsertAsset(model.Asset) error
	GetAsset(string) (model.Asset, error)
	IfAssetExists(string) bool
	UpdateAssetStatus(string, int32, string, bool) error
	UpdateAssetReady(string, bool) error
	UpdateStreamURL(string, string) error
	UpdateThumbnail(string, string) error
}

type assetDatabase struct {
	db DatabaseHelper
}

// NewAssetDatabase returns an instance of AssetDatabase.
func NewAssetDatabase(db DatabaseHelper) AssetDatabase {
	return &assetDatabase{
		db: db,
	}
}

func (a *assetDatabase) InsertAsset(asset model.Asset) error {
	insertResult, err := a.db.Collection("asset").InsertOne(context.Background(), asset)
	if err != nil {
		log.Error("Inserting an asset: ", err)
		return err
	}
	log.Info("Inserted an asset: ", insertResult)
	return nil
}

func (a *assetDatabase) GetAsset(assetID string) (model.Asset, error) {
	result := model.Asset{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := a.db.Collection("asset").FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error("Getting asset: ", err)
		return result, err
	}
	log.Info("Getting asset: ", result.AssetID)
	return result, nil
}

func (a *assetDatabase) IfAssetExists(assetID string) bool {
	result := model.Asset{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := a.db.Collection("asset").FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error("Checking if asset exists: ", err)
		return false
	}
	log.Info("Checking if asset exists: ", true)
	return true
}

func (a *assetDatabase) UpdateAssetStatus(assetID string, assetStatusCode int32, assetStatus string, assetError bool) error {
	filter := bson.M{"_id": assetID}
	update := bson.M{"$set": bson.M{
		"asset_status_code": assetStatusCode,
		"asset_status":      assetStatus,
		"asset_error":       assetError,
	}}
	result, err := a.db.Collection("asset").UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error("Updating asset status: ", err)
		return err
	}
	log.Info("Updated asset status: ", result)
	return nil
}

func (a *assetDatabase) UpdateAssetReady(assetID string, assetReady bool) error {
	filter := bson.M{"_id": assetID}
	update := bson.M{"$set": bson.M{
		"asset_ready": assetReady,
	}}
	result, err := a.db.Collection("asset").UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error("Updating asset ready status: ", err)
		return err
	}
	log.Info("Updating asset ready status: ", result)
	return nil
}

func (a *assetDatabase) UpdateStreamURL(assetID string, streamURL string) error {
	filter := bson.M{"_id": assetID}
	update := bson.M{"$set": bson.M{
		"stream_url": streamURL,
	}}
	result, err := a.db.Collection("asset").UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error("Updating streamURL: ", err)
		return err
	}
	log.Info("Updating streamURL: ", result)
	return nil
}

func (a *assetDatabase) UpdateThumbnail(assetID string, thumbnail string) error {
	filter := bson.M{"_id": assetID}
	update := bson.M{"$set": bson.M{
		"thumbnail": thumbnail,
	}}
	result, err := a.db.Collection("asset").UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error("Updating thumbnail: ", result)
		return err
	}
	log.Info("Updating thumbnail: ", result)
	return nil
}
