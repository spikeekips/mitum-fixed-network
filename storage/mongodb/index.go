package mongodbstorage

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var IndexPrefix = "mitum_"

var manifestIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "height", Value: 1}},
		Options: options.Index().
			SetName(indexName("manifest_height")).
			SetUnique(true),
	},
}

var operationIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "fact", Value: 1}},
		Options: options.Index().
			SetName(indexName("operation_fact")).
			SetUnique(true),
	},
	{
		Keys: bson.D{bson.E{Key: "height", Value: 1}},
		Options: options.Index().
			SetName(indexName("operation_height")),
	},
}

var stateIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "key", Value: 1}, bson.E{Key: "height", Value: 1}},
		Options: options.Index().
			SetName(indexName("state_key_and_height")).
			SetUnique(true),
	},
	{
		Keys: bson.D{bson.E{Key: "height", Value: 1}},
		Options: options.Index().
			SetName(indexName("state_height")),
	},
}

var proposalIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "hash_string", Value: 1}},
		Options: options.Index().
			SetName(indexName("proposal_hash")).
			SetUnique(true),
	},
	{
		Keys: bson.D{bson.E{Key: "height", Value: 1}, bson.E{Key: "round", Value: 1}, bson.E{Key: "proposer", Value: 1}},
		Options: options.Index().
			SetName(indexName("proposal_height_round_proposer")),
	},
}

var stagedOperationIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "hash", Value: 1}},
		Options: options.Index().
			SetName(indexName("hash")).
			SetUnique(true),
	},
	{
		Keys: bson.D{bson.E{Key: "inserted_at", Value: -1}},
		Options: options.Index().
			SetName(indexName("operation_inserted_at")),
	},
}

var voteproofIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "height", Value: 1}},
		Options: options.Index().
			SetName(indexName("voteproof_height")),
	},
	{
		Keys: bson.D{bson.E{Key: "height", Value: 1}, bson.E{Key: "stage", Value: 1}},
		Options: options.Index().
			SetName(indexName("voteproof_height_and_stage")).
			SetUnique(true),
	},
}

var blockdataMapIndexModels = []mongo.IndexModel{
	{
		Keys: bson.D{bson.E{Key: "height", Value: 1}},
		Options: options.Index().
			SetName(indexName("blockdata_map_height")).
			SetUnique(true),
	},
}

var defaultIndexes = map[string] /* collection */ []mongo.IndexModel{
	ColNameManifest:        manifestIndexModels,
	ColNameOperation:       operationIndexModels,
	ColNameProposal:        proposalIndexModels,
	ColNameStagedOperation: stagedOperationIndexModels,
	ColNameState:           stateIndexModels,
	ColNameVoteproof:       voteproofIndexModels,
	ColNameBlockdataMap:    blockdataMapIndexModels,
}

func indexName(s string) string {
	return fmt.Sprintf("%s%s", IndexPrefix, s)
}

func isIndexName(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}
