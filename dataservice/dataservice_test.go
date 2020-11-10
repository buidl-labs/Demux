package dataservice_test

import (
	"errors"
	"testing"

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/dataservice/mocks"

	"github.com/stretchr/testify/assert"
)

var dbName = "demux"
var uri = "mongodb://127.0.0.1:27017"

// TestNewDatabase tests new database creation.
func TestNewDatabase(t *testing.T) {
	dbClient, err := dataservice.NewClient(uri)
	assert.NoError(t, err)

	db := dataservice.NewDatabase(dbName, dbClient)

	assert.NotEmpty(t, db)
}

// TestStartSession tests starting session.
func TestStartSession(t *testing.T) {
	// The code below does not fall under the cover tool as we are testing mocks
	// and in order to test the actual code, we would need to expose internal
	// structures and create interfaces for them. In addition to this it would
	// require to mock them as well.

	// Of course we can use this approach to achieve 100% coverage but it is not
	// actually worth it to test mongo functionality itself. For such cases it
	// is better to use integration tests, but thats another topic.

	var db dataservice.DatabaseHelper
	var client dataservice.ClientHelper

	// db = &MockDatabaseHelper{} // can be used as db = &mocks.DatabaseHelper{}
	db = &mocks.DatabaseHelper{}
	client = &mocks.ClientHelper{}

	client.(*mocks.ClientHelper).On("StartSession").Return(nil, errors.New("mocked-error"))

	db.(*mocks.DatabaseHelper).On("Client").Return(client)

	// As we do not actual start any session then we do not need to check it.
	// It is possible to mock session interface and check for custom conditions
	// But this creates huge overhead to the unnecessary functionality.
	_, err := db.Client().StartSession()

	assert.EqualError(t, err, "mocked-error")
}
