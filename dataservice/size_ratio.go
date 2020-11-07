package dataservice

import (
	"context"

	"github.com/buidl-labs/Demux/model"

	log "github.com/sirupsen/logrus"
)

// SizeRatioDatabase provides the sizeRatio db operations.
type SizeRatioDatabase interface {
	InsertSizeRatio(model.SizeRatio) error
}

type sizeRatioDatabase struct {
	db DatabaseHelper
}

// NewSizeRatioDatabase returns an instance of SizeRatioDatabase.
func NewSizeRatioDatabase(db DatabaseHelper) SizeRatioDatabase {
	return &sizeRatioDatabase{
		db: db,
	}
}

func (sr *sizeRatioDatabase) InsertSizeRatio(sizeRatio model.SizeRatio) error {
	insertResult, err := sr.db.Collection("sizeRatio").InsertOne(context.Background(), sizeRatio)
	if err != nil {
		log.Error("Inserting a sizeRatio:", err)
		return err
	}
	log.Info("Inserted a sizeRatio: ", insertResult)
	return nil
}
