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
		log.Fatal(err)
	}

	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Connected to MongoDB!")

	assetCollection = client.Database(dbName).Collection("asset")
	uploadCollection = client.Database(dbName).Collection("upload")
	transcodingDealCollection = client.Database(dbName).Collection("transcodingDeal")
	storageDealCollection = client.Database(dbName).Collection("storageDeal")
	userCollection = client.Database(dbName).Collection("user")
	sizeRatioCollection = client.Database(dbName).Collection("sizeRatio")
	meanSizeRatioCollection = client.Database(dbName).Collection("meanSizeRatio")

	log.Info("Collections created!")
}

// InsertAsset inserts an asset in the DB
func InsertAsset(asset model.Asset) {
	insertResult, err := assetCollection.InsertOne(context.Background(), asset)
	if err != nil {
		log.Error("Inserting an asset", err)
	}
	log.Info("Inserted a Single Record ", insertResult.InsertedID)
}

// InsertUpload inserts an upload in the DB
func InsertUpload(upload model.Upload) {
	insertResult, err := uploadCollection.InsertOne(context.Background(), upload)
	if err != nil {
		log.Error("Inserting an upload:", err)
	}
	log.Info("Inserted a Single Record ", insertResult.InsertedID)
}

// InsertTranscodingDeal inserts a transcodingDeal in the DB
func InsertTranscodingDeal(transcodingDeal model.TranscodingDeal) {
	insertResult, err := transcodingDealCollection.InsertOne(context.Background(), transcodingDeal)
	if err != nil {
		log.Error("Inserting a transcodingDeal:", err)
	}
	log.Info("Inserted a Single Record ", insertResult.InsertedID)
}

// InsertStorageDeal inserts a storageDeal in the DB
func InsertStorageDeal(storageDeal model.StorageDeal) {
	insertResult, err := storageDealCollection.InsertOne(context.Background(), storageDeal)

	if err != nil {
		log.Error("Inserting a storageDeal:", err)
	}

	log.Info("Inserted a Single Record ", insertResult.InsertedID)
}

// InsertUser inserts a storageDeal in the DB
func InsertUser(user model.User) {
	insertResult, err := userCollection.InsertOne(context.Background(), user)
	if err != nil {
		log.Error("Inserting a user:", err)
	}
	log.Info("Inserted a Single Record ", insertResult.InsertedID)
}

// InsertSizeRatio inserts a sizeRatio in the DB
func InsertSizeRatio(sizeRatio model.SizeRatio) {
	insertResult, err := sizeRatioCollection.InsertOne(context.Background(), sizeRatio)
	if err != nil {
		log.Error("Inserting a sizeRatio:", err)
	}
	log.Info("Inserted a Single Record ", insertResult.InsertedID)
}

// InsertMeanSizeRatio inserts an upload in the DB
func InsertMeanSizeRatio(meanSizeRatio model.SizeRatio) {
	insertResult, err := meanSizeRatioCollection.InsertOne(context.Background(), meanSizeRatio)
	if err != nil {
		log.Error("Inserting a meanSizeRatio:", err)
	}
	log.Info("Inserted a Single Record ", insertResult.InsertedID)
}

// UpdateAssetStatus updates the status of an asset.
func UpdateAssetStatus(assetID string, assetStatusCode uint32, assetStatus string, assetError bool) {
	filter := bson.M{"_id": assetID}
	update := bson.M{"$set": bson.M{
		"asset_status_code": assetStatusCode,
		"asset_status":      assetStatus,
		"asset_error":       assetError,
	}}
	result, err := assetCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error(err)
	}
	log.Info("modified count:", result.ModifiedCount)
}

// UpdateUploadStatus updates the status of an upload.
func UpdateUploadStatus(assetID string, status bool) {
	filter := bson.M{"_id": assetID}
	update := bson.M{"$set": bson.M{
		"status": status,
	}}
	result, err := uploadCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error(err)
	}
	log.Info("modified count:", result.ModifiedCount)
}

// IfAssetExists checks whether a given asset exists in the database.
func IfAssetExists(assetID string) bool {
	result := model.Asset{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := assetCollection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error(err)
		return false
	}
	log.Info("got asset", result)
	return true
}

// IfUploadExists checks whether a given upload exists in the database.
func IfUploadExists(assetID string) bool {
	result := model.Upload{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := uploadCollection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error(err)
		return false
	}
	log.Info("got upload", result)
	return true
}

// GetAsset returns an asset.
func GetAsset(assetID string) model.Asset {
	result := model.Asset{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := assetCollection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error(err)
	}
	log.Info("got", result)
	return result
}

// GetUpload returns an upload.
func GetUpload(assetID string) model.Upload {
	result := model.Upload{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := uploadCollection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error(err)
	}
	log.Info("got", result)
	return result
}

// GetTranscodingDeal returns a transcoding deal.
func GetTranscodingDeal(assetID string) model.TranscodingDeal {
	result := model.TranscodingDeal{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := transcodingDealCollection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error(err)
	}
	log.Info("got", result)
	return result
}

// GetStorageDeal returns a storage deal.
func GetStorageDeal(assetID string) model.StorageDeal {
	result := model.StorageDeal{}
	filter := bson.D{primitive.E{Key: "_id", Value: assetID}}
	err := storageDealCollection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error(err)
	}
	log.Info("got", result)
	return result
}

// GetPendingDeals returns the pending storage deals.
func GetPendingDeals() []model.StorageDeal {
	filter := bson.D{primitive.E{Key: "storage_status_code", Value: 0}}
	cur, err := storageDealCollection.Find(context.Background(), filter)
	if err != nil {
		log.Error(err)
	}

	var results []model.StorageDeal
	for cur.Next(context.Background()) {
		var result model.StorageDeal
		e := cur.Decode(&result)
		if e != nil {
			log.Error(e)
		}
		results = append(results, result)
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	cur.Close(context.Background())
	return results
}

// UpdateStorageDeal updates a storage deal.
func UpdateStorageDeal(CID string, storageStatusCode uint32, storageStatus string, miner string, storageCost string, filecoinDealExpiry int64) {
	filter := bson.M{"cid": CID}
	update := bson.M{"$set": bson.M{
		"storage_status_code":  storageStatusCode,
		"storage_status":       storageStatus,
		"miner":                miner,
		"storage_cost":         storageCost,
		"filecoin_deal_expiry": filecoinDealExpiry,
	}}
	result, err := assetCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error(err)
	}
	log.Info("modified count:", result.ModifiedCount)
}

// UpdateStreamURL updates the StreamURL of an asset.
func UpdateStreamURL(assetID string, streamURL string) {
	filter := bson.M{"_id": assetID}
	update := bson.M{"$set": bson.M{
		"stream_url": streamURL,
	}}
	result, err := assetCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error(err)
	}
	log.Info("modified count:", result.ModifiedCount)
}

// UpdateThumbnail updates the thumbnail of an asset.
func UpdateThumbnail(assetID string, thumbnail string) {
	filter := bson.M{"_id": assetID}
	update := bson.M{"$set": bson.M{
		"thumbnail": thumbnail,
	}}
	result, err := assetCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error(err)
	}
	log.Info("modified count:", result.ModifiedCount)
}

// IfUserExists checks whether a user having a given digest exists in the database.
func IfUserExists(digest string) bool {
	result := model.User{}
	filter := bson.D{primitive.E{Key: "digest", Value: digest}}
	err := userCollection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error(err)
		return false
	}
	log.Info("got user", result)
	return true
}

// IncrementUserAssetCount increments a user's AssetCount by 1.
func IncrementUserAssetCount(digest string) {
	result, err := userCollection.UpdateOne(context.Background(), bson.M{
		"digest": digest,
	}, bson.D{
		primitive.E{Key: "$inc", Value: bson.D{primitive.E{Key: "asset_count", Value: 1}}},
	}, options.Update().SetUpsert(true))
	if err != nil {
		log.Error(err)
	}
	log.Info("modified count:", result.ModifiedCount)
}

// UpdateAssetReady updates the "ready" state of an asset.
func UpdateAssetReady(assetID string, assetReady bool) {
	filter := bson.M{"_id": assetID}
	update := bson.M{"$set": bson.M{
		"asset_ready": assetReady,
	}}
	result, err := assetCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error(err)
	}
	log.Info("modified count:", result.ModifiedCount)
}

// AddSizeRatio adds a new SizeRatio.
func AddSizeRatio(x model.SizeRatio) {
	insertResult, err := sizeRatioCollection.InsertOne(context.Background(), x)
	if err != nil {
		log.Error("Inserting a sizeRatio", err)
	}
	log.Info("Inserted a Single Record ", insertResult.InsertedID)
}

// UpdateMeanSizeRatio updates the mean size ratio .
func UpdateMeanSizeRatio(ratio float64, ratioSum float64, count uint64) {
	filter := bson.M{"_id": 1}
	update := bson.M{"$set": bson.M{
		"ratio":     ratio,
		"ratio_sum": ratioSum,
		"count":     count,
	}}
	result, err := meanSizeRatioCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		log.Error("UpdateMeanSizeRatio: ", err)
		return
	}
	log.Info("UpdateMeanSizeRatio modified count: ", result.ModifiedCount)
}

// GetMeanSizeRatio returns the current mean size ratio.
func GetMeanSizeRatio() model.MeanSizeRatio {
	result := model.MeanSizeRatio{}
	filter := bson.D{primitive.E{Key: "_id", Value: 1}}
	err := meanSizeRatioCollection.FindOne(context.Background(), filter).Decode(&result)
	if err != nil {
		log.Error(err)
	}
	log.Info("GetMeanSizeRatio ", result)
	return result
}
