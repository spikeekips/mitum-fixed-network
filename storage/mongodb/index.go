package mongodbstorage

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var blockIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "height", Value: 1}},
		Options: options.Index().
			SetName("block_height").
			SetUnique(true),
	},
}

var manifestIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "height", Value: 1}},
		Options: options.Index().
			SetName("manifest_height").
			SetUnique(true),
	},
}

var operationIndexModels []mongo.IndexModel

var proposalIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "height", Value: 1}, bson.E{Key: "round", Value: 1}},
		Options: options.Index().
			SetName("proposal_height_and_round"),
	},
}

var sealIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "inserted_at", Value: -1}},
		Options: options.Index().
			SetName("seal_inserted_at"),
	},
}

var voteproofIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "height", Value: -1}},
		Options: options.Index().
			SetName("voteproof_height"),
	},
	{
		Keys: bson.D{bson.E{Key: "stage", Value: 1}},
		Options: options.Index().
			SetName("voteproof_stage"),
	},
	{
		Keys: bson.D{bson.E{Key: "height", Value: -1}, bson.E{Key: "stage", Value: 1}},
		Options: options.Index().
			SetName("voteproof_height_and_stage"),
	},
}

var defaultIndexes = map[string] /* collection */ []mongo.IndexModel{
	"block":     blockIndexModels,
	"manifest":  manifestIndexModels,
	"operation": operationIndexModels,
	"proposal":  proposalIndexModels,
	"seal":      sealIndexModels,
	"voteproof": voteproofIndexModels,
}
