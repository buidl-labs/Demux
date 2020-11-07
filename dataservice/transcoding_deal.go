package dataservice

import (
	"context"

	"github.com/buidl-labs/Demux/model"

	log "github.com/sirupsen/logrus"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TranscodingDealDatabase provides the transcodingDeal db operations.
type TranscodingDealDatabase interface {
	InsertTranscodingDeal(model.TranscodingDeal) error
	GetTranscodingDeal(string) (model.TranscodingDeal, error)
}

type transcodingDealDatabase struct {
	db DatabaseHelper
}

// NewTranscodingDealDatabase returns an instance of TranscodingDealDatabase.
func NewTranscodingDealDatabase(db DatabaseHelper) TranscodingDealDatabase {
	return &transcodingDealDatabase{
		db: db,
	}
}

func (t *transcodingDealDatabase) InsertTranscodingDeal(transcodingDeal model.TranscodingDeal) error {
	insertResult, err := t.db.Collection("transcodingDeal").InsertOne(context.Background(), transcodingDeal)
	if err != nil {
		log.Error("Inserting a transcodingDeal: ", err)
		return err
	}
	log.Info("Inserted a transcodingDeal: ", insertResult)
	return nil
}

func (t *transcodingDealDatabase) GetTranscodingDeal(assetID string) (model.TranscodingDeal, error) {
	result := model.TranscodingDeal{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := t.db.Collection("transcodingDeal").FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Warn("Getting transcodingDeal: ", err)
		return result, err
	}
	log.Info("Getting transcodingDeal: ", result.AssetID)
	return result, nil
}
