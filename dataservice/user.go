package dataservice

import (
	"context"

	"github.com/buidl-labs/Demux/model"

	log "github.com/sirupsen/logrus"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserDatabase provides the user db operations.
type UserDatabase interface {
	IfUserExists(string) bool
	IncrementUserAssetCount(string) error
}

type userDatabase struct {
	db DatabaseHelper
}

// NewUserDatabase returns an instance of UserDatabase.
func NewUserDatabase(db DatabaseHelper) UserDatabase {
	return &userDatabase{
		db: db,
	}
}

func (u *userDatabase) IfUserExists(digest string) bool {
	result := model.User{}
	filter := bson.D{primitive.E{Key: "digest", Value: digest}}
	err := u.db.Collection("user").FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error("Checking if user exists: ", err)
		return false
	}
	log.Info("Checking if user exists: ", true)
	return true
}

// IncrementUserAssetCount increments a user's AssetCount by 1.
func (u *userDatabase) IncrementUserAssetCount(digest string) error {
	result, err := u.db.Collection("user").UpdateOne(context.Background(), bson.M{
		"digest": digest,
	}, bson.D{
		primitive.E{Key: "$inc", Value: bson.D{primitive.E{Key: "asset_count", Value: 1}}},
	})
	if err != nil {
		log.Error("Incrementing user asset count: ", err)
		return err
	}
	log.Info("Incrementing user asset count: ", result)
	return nil
}
