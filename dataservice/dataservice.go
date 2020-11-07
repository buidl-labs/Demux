package dataservice

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"

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

// DatabaseHelper is an abstraction of the mongodb client instance.
type DatabaseHelper interface {
	Collection(name string) CollectionHelper
	Client() ClientHelper
}

// CollectionHelper provides access to basic mongodb collection operations.
type CollectionHelper interface {
	FindOne(context.Context, interface{}) SingleResultHelper
	Find(ctx context.Context, filter interface{}) (*mongo.Cursor, error)
	InsertOne(context.Context, interface{}) (interface{}, error)
	UpdateOne(context.Context, interface{}, interface{}) (int64, error)
	DeleteOne(ctx context.Context, filter interface{}) (int64, error)
}

// SingleResultHelper provides access to the mongo Decode function for a single result.
type SingleResultHelper interface {
	Decode(v interface{}) error
}

// ClientHelper provides access to methods to control the db client.
type ClientHelper interface {
	Database(string) DatabaseHelper
	Connect() error
	StartSession() (mongo.Session, error)
}

type mongoClient struct {
	cl *mongo.Client
}

type mongoDatabase struct {
	db *mongo.Database
}

type mongoCollection struct {
	coll *mongo.Collection
}

type mongoSingleResult struct {
	sr *mongo.SingleResult
}

type mongoSession struct {
	mongo.Session
}

// NewClient returns a ClientHelper for a given MONGO_URI.
func NewClient(uri string) (ClientHelper, error) {
	c, err := mongo.NewClient(options.Client().ApplyURI(uri))
	return &mongoClient{cl: c}, err
}

// NewDatabase returns a DatabaseHelper for a given db name and ClientHelper.
func NewDatabase(dbName string, client ClientHelper) DatabaseHelper {
	return client.Database(dbName)
}

func (mc *mongoClient) Database(dbName string) DatabaseHelper {
	db := mc.cl.Database(dbName)
	return &mongoDatabase{db: db}
}

func (mc *mongoClient) StartSession() (mongo.Session, error) {
	session, err := mc.cl.StartSession()
	return &mongoSession{session}, err
}

func (mc *mongoClient) Connect() error {
	// mongo client does not use context on connect method. There is a ticket
	// with a request to deprecate this functionality and another one with
	// explanation why it could be useful in synchronous requests.
	// https://jira.mongodb.org/browse/GODRIVER-1031
	// https://jira.mongodb.org/browse/GODRIVER-979
	return mc.cl.Connect(nil)
}

func (md *mongoDatabase) Collection(colName string) CollectionHelper {
	collection := md.db.Collection(colName)
	return &mongoCollection{coll: collection}
}

func (md *mongoDatabase) Client() ClientHelper {
	client := md.db.Client()
	return &mongoClient{cl: client}
}

func (mc *mongoCollection) FindOne(ctx context.Context, filter interface{}) SingleResultHelper {
	singleResult := mc.coll.FindOne(ctx, filter)
	return &mongoSingleResult{sr: singleResult}
}

func (mc *mongoCollection) Find(ctx context.Context, filter interface{}) (*mongo.Cursor, error) {
	cur, err := mc.coll.Find(ctx, filter)
	return cur, err
}

func (mc *mongoCollection) InsertOne(ctx context.Context, document interface{}) (interface{}, error) {
	id, err := mc.coll.InsertOne(ctx, document)
	return id.InsertedID, err
}

func (mc *mongoCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (int64, error) {
	result, err := mc.coll.UpdateOne(ctx, filter, update)
	return result.ModifiedCount, err
}

func (mc *mongoCollection) DeleteOne(ctx context.Context, filter interface{}) (int64, error) {
	count, err := mc.coll.DeleteOne(ctx, filter)
	return count.DeletedCount, err
}

func (sr *mongoSingleResult) Decode(v interface{}) error {
	return sr.sr.Decode(v)
}

// InitMongoClient initializes the mongo client.
func InitMongoClient() DatabaseHelper {

	client, err := NewClient(connectionString)
	if err != nil {
		log.Fatal("mdb conn: ", err)
	}
	err = client.Connect()
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	db := NewDatabase("demux", client)

	return db

}
