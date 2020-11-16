package dataservice_test

import (
	"testing"

	"github.com/buidl-labs/Demux/dataservice"
	"github.com/buidl-labs/Demux/dataservice/mocks"
	"github.com/buidl-labs/Demux/model"

	"github.com/stretchr/testify/assert"
)

func TestNewAssetDatabase(t *testing.T) {
	dbClient, err := dataservice.NewClient(uri)
	assert.NoError(t, err)

	db := dataservice.NewDatabase(dbName, dbClient)

	assetDB := dataservice.NewAssetDatabase(db)

	assert.NotEmpty(t, assetDB)
}

func TestGetAsset(t *testing.T) {
	assetDba := &mocks.AssetDatabase{}

	asset1, err := assetDba.GetAsset("1b2e976a-983d-4845-967a-f60b33c82869")
	assert.NoError(t, err)
	assert.Equal(t, model.Asset{
		AssetID:         "1b2e976a-983d-4845-967a-f60b33c82869",
		AssetReady:      false,
		AssetStatusCode: 0,
		AssetStatus:     "asset created",
		AssetError:      false,
		CreatedAt:       1605030069,
		Thumbnail:       "",
	}, asset1)

	asset2, err := assetDba.GetAsset("some-wrong-asset-id")
	assert.EqualError(t, err, "couldn't find asset")
	assert.Equal(t, model.Asset{}, asset2)
}

func TestInsertAsset(t *testing.T) {
	assetDba := &mocks.AssetDatabase{}

	err := assetDba.InsertAsset(model.Asset{
		AssetID:         "1b2e976a-983d-4845-967a-f60b33c82869",
		AssetReady:      false,
		AssetStatusCode: 0,
		AssetStatus:     "asset created",
		AssetError:      false,
		CreatedAt:       1605030069,
		Thumbnail:       "",
	})
	assert.NoError(t, err)
}
