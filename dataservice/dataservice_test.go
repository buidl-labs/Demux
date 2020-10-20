package dataservice_test

import (
	"context"
	"os"
	"testing"
	"time"

	// "github.com/buidl-labs/Demux/internal"
	// "github.com/buidl-labs/Demux/model"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	dbName = "demux"
	// connectionString          = os.Getenv("MONGO_URI")
	connectionString          = "mongodb://127.0.0.1:27017/demux"
	assetCollection           *mongo.Collection
	uploadCollection          *mongo.Collection
	transcodingDealCollection *mongo.Collection
	storageDealCollection     *mongo.Collection
	userCollection            *mongo.Collection
	sizeRatioCollection       *mongo.Collection
	meanSizeRatioCollection   *mongo.Collection
)

func TestMain(m *testing.M) {
	// log.Println("Do stuff BEFORE the tests!")
	exitVal := m.Run()
	// log.Println("Do stuff AFTER the tests!")

	os.Exit(exitVal)
}

func requireCursorLength(t *testing.T, cursor *mongo.Cursor, length int) {
	i := 0
	for cursor.Next(context.Background()) {
		i++
	}

	require.NoError(t, cursor.Err())
	require.Equal(t, i, length)
}

func getCollection(t *testing.T, collName string) *mongo.Collection {
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// cs := testutil.ConnString(t)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(connectionString))
	require.NoError(t, err)
	// defer client.Disconnect(ctx)

	db := client.Database(dbName)
	coll := db.Collection(collName)
	return coll
}

func TestInsertAsset(t *testing.T) {
	assetCollection := getCollection(t, "asset")

	err := assetCollection.Drop(context.Background())
	require.NoError(t, err)

	{
		// Start Example 1
		result, err := assetCollection.InsertOne(
			context.Background(),
			bson.D{
				primitive.E{Key: "_id", Value: "24fe2218-a969-4e19-8c0d-98238fc6a0c9"},
				primitive.E{Key: "asset_ready", Value: false},
				primitive.E{Key: "asset_status_code", Value: -1},
				primitive.E{Key: "asset_status", Value: "asset created"},
				primitive.E{Key: "asset_error", Value: false},
				primitive.E{Key: "stream_url", Value: ""},
				primitive.E{Key: "thumbnail", Value: ""},
				primitive.E{Key: "created_at", Value: 1982827332},
			})

		// End Example 1
		require.NoError(t, err)
		require.Equal(t, "24fe2218-a969-4e19-8c0d-98238fc6a0c9", result.InsertedID, "_id should be equal")
		require.NotNil(t, result.InsertedID)
	}

	{
		// Start Example 2
		cursor, err := assetCollection.Find(
			context.Background(),
			bson.D{
				primitive.E{Key: "_id", Value: "24fe2218-a969-4e19-8c0d-98238fc6a0c9"},
				primitive.E{Key: "asset_ready", Value: false},
				primitive.E{Key: "asset_status_code", Value: -1},
				primitive.E{Key: "asset_status", Value: "asset created"},
				primitive.E{Key: "asset_error", Value: false},
				primitive.E{Key: "stream_url", Value: ""},
				primitive.E{Key: "thumbnail", Value: ""},
				primitive.E{Key: "created_at", Value: 1982827332},
			})

		// End Example 2
		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}
}

func TestInsertUpload(t *testing.T) {
	uploadCollection := getCollection(t, "upload")

	err := uploadCollection.Drop(context.Background())
	require.NoError(t, err)

	{
		// Start Example 1
		result, err := uploadCollection.InsertOne(
			context.Background(),
			bson.D{
				primitive.E{Key: "_id", Value: "24fe2218-a969-4e19-8c0d-98238fc6a0c9"},
				primitive.E{Key: "url", Value: "http://localhost:8000/fileupload/24fe2218-a969-4e19-8c0d-98238fc6a0c9"},
				primitive.E{Key: "status", Value: false},
				primitive.E{Key: "error", Value: false},
			})

		// End Example 1
		require.NoError(t, err)
		require.Equal(t, "24fe2218-a969-4e19-8c0d-98238fc6a0c9", result.InsertedID, "_id should be equal")
		require.NotNil(t, result.InsertedID)
	}

	{
		// Start Example 2
		cursor, err := uploadCollection.Find(
			context.Background(),
			bson.D{
				primitive.E{Key: "_id", Value: "24fe2218-a969-4e19-8c0d-98238fc6a0c9"},
				primitive.E{Key: "url", Value: "http://localhost:8000/fileupload/24fe2218-a969-4e19-8c0d-98238fc6a0c9"},
				primitive.E{Key: "status", Value: false},
				primitive.E{Key: "error", Value: false},
			})

		// End Example 2
		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}
}

func TestSizeRatio(t *testing.T) {
	sizeRatioCollection := getCollection(t, "sizeRatio")

	err := sizeRatioCollection.Drop(context.Background())
	require.NoError(t, err)

	{
		// Start Example 1
		result, err := sizeRatioCollection.InsertOne(
			context.Background(),
			bson.D{
				primitive.E{Key: "_id", Value: "24fe2218-a969-4e19-8c0d-98238fc6a0c9"},
				primitive.E{Key: "size_ratio", Value: 2.5},
				primitive.E{Key: "video_file_fize", Value: 10},
				primitive.E{Key: "stream_folder_size", Value: 25},
			})

		// End Example 1
		require.NoError(t, err)
		require.Equal(t, "24fe2218-a969-4e19-8c0d-98238fc6a0c9", result.InsertedID, "_id should be equal")
		require.NotNil(t, result.InsertedID)
	}

	{
		// Start Example 2
		cursor, err := sizeRatioCollection.Find(
			context.Background(),
			bson.D{
				primitive.E{Key: "_id", Value: "24fe2218-a969-4e19-8c0d-98238fc6a0c9"},
				primitive.E{Key: "size_ratio", Value: 2.5},
				primitive.E{Key: "video_file_fize", Value: 10},
				primitive.E{Key: "stream_folder_size", Value: 25},
			})

		// End Example 2
		require.NoError(t, err)
		requireCursorLength(t, cursor, 1)
	}
}

// TODO
// func TestInsertStorageDeal(t *testing.T) {
// 	storageDealCollection := getCollection(t, "storageDeal")

// 	err := storageDealCollection.Drop(context.Background())
// 	require.NoError(t, err)

// 	{
// 		// Start Example 1
// 		result, err := storageDealCollection.InsertOne(
// 			context.Background(),
// 			bson.D{
// 				primitive.E{Key: "_id", Value: "24fe2218-a969-4e19-8c0d-98238fc6a0c9"},
// 				primitive.E{Key: "url", Value: "http://localhost:8000/fileupload/24fe2218-a969-4e19-8c0d-98238fc6a0c9"},
// 				primitive.E{Key: "status", Value: false},
// 				primitive.E{Key: "error", Value: false},
// 			})

// 		// End Example 1
// 		require.NoError(t, err)
// 		require.Equal(t, "24fe2218-a969-4e19-8c0d-98238fc6a0c9", result.InsertedID, "_id should be equal")
// 		require.NotNil(t, result.InsertedID)
// 	}

// 	{
// 		// Start Example 2
// 		cursor, err := storageDealCollection.Find(
// 			context.Background(),
// 			bson.D{
// 				primitive.E{Key: "_id", Value: "24fe2218-a969-4e19-8c0d-98238fc6a0c9"},
// 				primitive.E{Key: "url", Value: "http://localhost:8000/fileupload/24fe2218-a969-4e19-8c0d-98238fc6a0c9"},
// 				primitive.E{Key: "status", Value: false},
// 				primitive.E{Key: "error", Value: false},
// 			})

// 		// End Example 2
// 		require.NoError(t, err)
// 		requireCursorLength(t, cursor, 1)
// 	}
// }

/*
func TestDataservice(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// cs := testutil.ConnString(t)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(connectionString))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	db := client.Database(dbName)

	assetCollection = db.Collection("asset")
	uploadCollection = db.Collection("upload")
	transcodingDealCollection = db.Collection("transcodingDeal")
	storageDealCollection = db.Collection("storageDeal")
	userCollection = db.Collection("user")
	sizeRatioCollection = db.Collection("sizeRatio")
	meanSizeRatioCollection = db.Collection("meanSizeRatio")

	// InsertAssetTests(t, db, assetCollection)
}
*/
