package dataservice

import (
	"context"
	"os"

	"github.com/buidl-labs/Demux/model"

	log "github.com/sirupsen/logrus"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	dbName                    = "demux"
	connectionString          = os.Getenv("MONGO_URI")
	assetCollection           *mongo.Collection
	uploadCollection          *mongo.Collection
	transcodingDealCollection *mongo.Collection
	storageDealCollection     *mongo.Collection
	userCollection            *mongo.Collection
	sizeRatioCollection       *mongo.Collection
	meanSizeRatioCollection   *mongo.Collection
)

// InitMongoClient initializes the mongo client.
func InitMongoClient() {
	// Set client options
	clientOptions := options.Client().ApplyURI(connectionString)

	// connect to MongoDB
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal("Connecting to MongoDB: ", err)
	}

	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal("Checking MongoDB connection: ", err)
	}

	log.Info("Connected to MongoDB! ✅")

	assetCollection = client.Database(dbName).Collection("asset")
	uploadCollection = client.Database(dbName).Collection("upload")
	transcodingDealCollection = client.Database(dbName).Collection("transcodingDeal")
	storageDealCollection = client.Database(dbName).Collection("storageDeal")
	userCollection = client.Database(dbName).Collection("user")
	sizeRatioCollection = client.Database(dbName).Collection("sizeRatio")
	meanSizeRatioCollection = client.Database(dbName).Collection("meanSizeRatio")

	log.Info("Collections created ✅")
}

// InsertAsset inserts an asset in the DB.
func InsertAsset(asset model.Asset) error {
	insertResult, err := assetCollection.InsertOne(context.Background(), asset)
	if err != nil {
		log.Error("Inserting an asset: ", err)
		return err
	}
	log.Info("Inserted an asset: ", insertResult.InsertedID)
	return nil
}

// InsertUpload inserts an upload in the DB.
func InsertUpload(upload model.Upload) error {
	insertResult, err := uploadCollection.InsertOne(context.Background(), upload)
	if err != nil {
		log.Error("Inserting an upload: ", err)
		return err
	}
	log.Info("Inserted an upload: ", insertResult.InsertedID)
	return nil
}

// InsertTranscodingDeal inserts a transcodingDeal in the DB.
func InsertTranscodingDeal(transcodingDeal model.TranscodingDeal) error {
	insertResult, err := transcodingDealCollection.InsertOne(context.Background(), transcodingDeal)
	if err != nil {
		log.Error("Inserting a transcodingDeal: ", err)
		return err
	}
	log.Info("Inserted a transcodingDeal: ", insertResult.InsertedID)
	return nil
}

// InsertStorageDeal inserts a storageDeal in the DB.
func InsertStorageDeal(storageDeal model.StorageDeal) error {
	insertResult, err := storageDealCollection.InsertOne(context.Background(), storageDeal)
	if err != nil {
		log.Error("Inserting a storageDeal:", err)
		return err
	}
	log.Info("Inserted a storageDeal: ", insertResult.InsertedID)
	return nil
}

// InsertUser inserts a storageDeal in the DB.
func InsertUser(user model.User) error {
	insertResult, err := userCollection.InsertOne(context.Background(), user)
	if err != nil {
		log.Error("Inserting a user:", err)
		return err
	}
	log.Info("Inserted a user: ", insertResult.InsertedID)
	return nil
}

// InsertSizeRatio inserts a sizeRatio in the DB.
func InsertSizeRatio(sizeRatio model.SizeRatio) error {
	insertResult, err := sizeRatioCollection.InsertOne(context.Background(), sizeRatio)
	if err != nil {
		log.Error("Inserting a sizeRatio:", err)
		return err
	}
	log.Info("Inserted a sizeRatio: ", insertResult.InsertedID)
	return nil
}

// InsertMeanSizeRatio inserts an upload in the DB.
func InsertMeanSizeRatio(meanSizeRatio model.SizeRatio) error {
	insertResult, err := meanSizeRatioCollection.InsertOne(context.Background(), meanSizeRatio)
	if err != nil {
		log.Error("Inserting a meanSizeRatio:", err)
		return err
	}
	log.Info("Inserted a meanSizeRatio: ", insertResult.InsertedID)
	return nil
}

// UpdateAssetStatus updates the status of an asset.
func UpdateAssetStatus(assetID string, assetStatusCode int32, assetStatus string, assetError bool) error {
	filter := bson.M{"_id": assetID}
	update := bson.M{"$set": bson.M{
		"asset_status_code": assetStatusCode,
		"asset_status":      assetStatus,
		"asset_error":       assetError,
	}}
	result, err := assetCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error("Updating asset status: ", err)
		return err
	}
	log.Info("Updated asset status: ", result.ModifiedCount)
	return nil
}

// UpdateUploadStatus updates the status of an upload.
func UpdateUploadStatus(assetID string, status bool) error {
	filter := bson.M{"_id": assetID}
	update := bson.M{"$set": bson.M{
		"status": status,
	}}
	result, err := uploadCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error("Updating upload status: ", err)
		return err
	}
	log.Info("Updating upload status: ", result.ModifiedCount)
	return nil
}

// IfAssetExists checks whether a given asset exists in the database.
func IfAssetExists(assetID string) bool {
	result := model.Asset{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := assetCollection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error("Checking if asset exists: ", err)
		return false
	}
	log.Info("Checking if asset exists: ", true)
	return true
}

// IfUploadExists checks whether a given upload exists in the database.
func IfUploadExists(assetID string) bool {
	result := model.Upload{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := uploadCollection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error("Checking if upload exists: ", err)
		return false
	}
	log.Info("Checking if upload exists: ", true)
	return true
}

// IfUserExists checks whether a given user exists in the database.
func IfUserExists(digest string) bool {
	result := model.User{}
	filter := bson.D{primitive.E{Key: "digest", Value: digest}}
	err := userCollection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error("Checking if user exists: ", err)
		return false
	}
	log.Info("Checking if user exists: ", true)
	return true
}

// GetAsset returns an asset.
func GetAsset(assetID string) (model.Asset, error) {
	result := model.Asset{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := assetCollection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error("Getting asset: ", err)
		return result, err
	}
	log.Info("Getting asset: ", result.AssetID)
	return result, nil
}

// GetUpload returns an upload.
func GetUpload(assetID string) (model.Upload, error) {
	result := model.Upload{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := uploadCollection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error("Getting upload: ", err)
		return result, err
	}
	log.Info("Getting upload: ", result.AssetID)
	return result, nil
}

// GetTranscodingDeal returns a transcoding deal.
func GetTranscodingDeal(assetID string) (model.TranscodingDeal, error) {
	result := model.TranscodingDeal{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := transcodingDealCollection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Warn("Getting transcodingDeal: ", err)
		return result, err
	}
	log.Info("Getting transcodingDeal: ", result.AssetID)
	return result, nil
}

// GetStorageDeal returns a storage deal.
func GetStorageDeal(assetID string) (model.StorageDeal, error) {
	result := model.StorageDeal{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := storageDealCollection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Warn("Getting storageDeal: ", err)
		return result, err
	}
	log.Info("Getting storageDeal: ", result.AssetID)
	return result, nil
}

// GetMeanSizeRatio returns the current mean size ratio.
func GetMeanSizeRatio() (model.MeanSizeRatio, error) {
	result := model.MeanSizeRatio{}
	filter := bson.D{primitive.E{Key: "_id", Value: 1}}
	err := meanSizeRatioCollection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error("Getting meanSizeRatio: ", err)
		return result, err
	}
	log.Info("Getting meanSizeRatio: ", result.ID)
	return result, nil
}

// GetPendingDeals returns the pending storage deals.
func GetPendingDeals() ([]model.StorageDeal, error) {
	var results = []model.StorageDeal{}

	filter := bson.D{primitive.E{Key: "storage_status_code", Value: 0}}
	cur, err := storageDealCollection.Find(context.Background(), filter)
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

// UpdateStorageDeal updates a storage deal.
func UpdateStorageDeal(CID string, storageStatusCode uint32, storageStatus string, miner string, storageCost string, filecoinDealExpiry int64) error {
	filter := bson.M{"cid": CID}
	update := bson.M{"$set": bson.M{
		"storage_status_code":  storageStatusCode,
		"storage_status":       storageStatus,
		"miner":                miner,
		"storage_cost":         storageCost,
		"filecoin_deal_expiry": filecoinDealExpiry,
	}}
	result, err := storageDealCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error("Updating storageDeal: ", err)
		return err
	}
	log.Info("Updating storageDeal: ", result.ModifiedCount)
	return nil
}

// UpdateStreamURL updates the StreamURL of an asset.
func UpdateStreamURL(assetID string, streamURL string) error {
	filter := bson.M{"_id": assetID}
	update := bson.M{"$set": bson.M{
		"stream_url": streamURL,
	}}
	result, err := assetCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error("Updating streamURL: ", err)
		return err
	}
	log.Info("Updating streamURL: ", result.ModifiedCount)
	return nil
}

// UpdateThumbnail updates the thumbnail of an asset.
func UpdateThumbnail(assetID string, thumbnail string) error {
	filter := bson.M{"_id": assetID}
	update := bson.M{"$set": bson.M{
		"thumbnail": thumbnail,
	}}
	result, err := assetCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error("Updating thumbnail: ", result)
		return err
	}
	log.Info("Updating thumbnail: ", result.ModifiedCount)
	return nil
}

// IncrementUserAssetCount increments a user's AssetCount by 1.
func IncrementUserAssetCount(digest string) error {
	result, err := userCollection.UpdateOne(context.Background(), bson.M{
		"digest": digest,
	}, bson.D{
		primitive.E{Key: "$inc", Value: bson.D{primitive.E{Key: "asset_count", Value: 1}}},
	}, options.Update().SetUpsert(true))
	if err != nil {
		log.Error("Incrementing user asset count: ", err)
		return err
	}
	log.Info("Incrementing user asset count: ", result.ModifiedCount)
	return nil
}

// UpdateAssetReady updates the "ready" state of an asset.
func UpdateAssetReady(assetID string, assetReady bool) error {
	filter := bson.M{"_id": assetID}
	update := bson.M{"$set": bson.M{
		"asset_ready": assetReady,
	}}
	result, err := assetCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error("Updating asset ready status: ", err)
		return err
	}
	log.Info("Updating asset ready status: ", result.ModifiedCount)
	return nil
}

// UpdateMeanSizeRatio updates the mean size ratio.
func UpdateMeanSizeRatio(ratio float64, ratioSum float64, count uint64) error {
	filter := bson.M{"_id": 1}
	update := bson.M{"$set": bson.M{
		"ratio":     ratio,
		"ratio_sum": ratioSum,
		"count":     count,
	}}
	result, err := meanSizeRatioCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error("Updating meanSizeRatio: ", err)
		return err
	}
	log.Info("Updating meanSizeRatio: ", result.ModifiedCount)
	return nil
}

// UpdateAssetStatusByCID updates the status of an asset.
func UpdateAssetStatusByCID(CID string, assetStatusCode int32, assetStatus string) error {
	storageDealResult := model.StorageDeal{}
	storageDealFilter := bson.D{primitive.E{Key: "cid", Value: CID}}
	err := storageDealCollection.FindOne(context.Background(), storageDealFilter).Decode(&storageDealResult)
	if err != nil {
		log.Warn("Getting storageDeal by CID: ", err)
		return err
	}
	log.Info("Getting storageDeal by CID: ", storageDealResult.AssetID)

	filter := bson.M{"_id": storageDealResult.AssetID}
	update := bson.M{"$set": bson.M{
		"asset_status_code": assetStatusCode,
		"asset_status":      assetStatus,
	}}
	result, err := assetCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error("Updating asset status by CID: ", err)
		return err
	}
	log.Info("Updated asset status by CID: ", result.ModifiedCount)
	return nil
}
