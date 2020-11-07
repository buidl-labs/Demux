package dataservice

import (
	"context"

	"github.com/buidl-labs/Demux/model"

	log "github.com/sirupsen/logrus"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UploadDatabase provides the upload db operations.
type UploadDatabase interface {
	InsertUpload(model.Upload) error
	GetUpload(string) (model.Upload, error)
	IfUploadExists(string) bool
	UpdateUploadStatus(string, bool) error
}

type uploadDatabase struct {
	db DatabaseHelper
}

// NewUploadDatabase returns an instance of UploadDatabase.
func NewUploadDatabase(db DatabaseHelper) UploadDatabase {
	return &uploadDatabase{
		db: db,
	}
}

func (up *uploadDatabase) InsertUpload(upload model.Upload) error {
	insertResult, err := up.db.Collection("upload").InsertOne(context.Background(), upload)
	if err != nil {
		log.Error("Inserting an upload: ", err)
		return err
	}
	log.Info("Inserted an upload: ", insertResult)
	return nil
}

func (up *uploadDatabase) GetUpload(assetID string) (model.Upload, error) {
	result := model.Upload{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := up.db.Collection("upload").FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error("Getting upload: ", err)
		return result, err
	}
	log.Info("Getting upload: ", result.AssetID)
	return result, nil
}

func (up *uploadDatabase) IfUploadExists(assetID string) bool {
	result := model.Upload{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := up.db.Collection("upload").FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error("Checking if upload exists: ", err)
		return false
	}
	log.Info("Checking if upload exists: ", true)
	return true
}

func (up *uploadDatabase) UpdateUploadStatus(assetID string, status bool) error {
	filter := bson.M{"_id": assetID}
	update := bson.M{"$set": bson.M{
		"status": status,
	}}
	result, err := up.db.Collection("upload").UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error("Updating upload status: ", err)
		return err
	}
	log.Info("Updating upload status: ", result)
	return nil
}
