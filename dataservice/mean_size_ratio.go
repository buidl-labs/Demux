package dataservice

import (
	"context"

	"github.com/buidl-labs/Demux/model"

	log "github.com/sirupsen/logrus"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MeanSizeRatioDatabase provides the msr db operations.
type MeanSizeRatioDatabase interface {
	InsertMeanSizeRatio(model.SizeRatio) error
	GetMeanSizeRatio() (model.MeanSizeRatio, error)
	UpdateMeanSizeRatio(float64, float64, uint64) error
}

type meanSizeRatioDatabase struct {
	db DatabaseHelper
}

// NewMeanSizeRatioDatabase returns an instance of MeanSizeRatioDatabase.
func NewMeanSizeRatioDatabase(db DatabaseHelper) MeanSizeRatioDatabase {
	return &meanSizeRatioDatabase{
		db: db,
	}
}

func (msr *meanSizeRatioDatabase) InsertMeanSizeRatio(meanSizeRatio model.SizeRatio) error {
	insertResult, err := msr.db.Collection("meanSizeRatio").InsertOne(context.Background(), meanSizeRatio)
	if err != nil {
		log.Error("Inserting a meanSizeRatio:", err)
		return err
	}
	log.Info("Inserted a meanSizeRatio: ", insertResult)
	return nil
}

func (msr *meanSizeRatioDatabase) GetMeanSizeRatio() (model.MeanSizeRatio, error) {
	result := model.MeanSizeRatio{}
	filter := bson.D{primitive.E{Key: "_id", Value: 1}}
	err := msr.db.Collection("meanSizeRatio").FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error("Getting meanSizeRatio: ", err)
		return result, err
	}
	log.Info("Getting meanSizeRatio: ", result.ID)
	return result, nil
}

func (msr *meanSizeRatioDatabase) UpdateMeanSizeRatio(ratio float64, ratioSum float64, count uint64) error {
	filter := bson.M{"_id": 1}
	update := bson.M{"$set": bson.M{
		"ratio":     ratio,
		"ratio_sum": ratioSum,
		"count":     count,
	}}
	result, err := msr.db.Collection("meanSizeRatio").UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error("Updating meanSizeRatio: ", err)
		return err
	}
	log.Info("Updating meanSizeRatio: ", result)
	return nil
}
