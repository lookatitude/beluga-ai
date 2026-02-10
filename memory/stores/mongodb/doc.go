// Package mongodb provides a MongoDB-backed implementation of [memory.MessageStore].
// Messages are stored as BSON documents in a MongoDB collection with a
// monotonically increasing sequence field for chronological ordering.
//
// # Usage
//
//	import "github.com/lookatitude/beluga-ai/memory/stores/mongodb"
//
//	store, err := mongodb.New(mongodb.Config{
//	    Collection: client.Database("beluga").Collection("messages"),
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	err = store.Append(ctx, msg)
//	results, err := store.Search(ctx, "query", 10)
//	all, err := store.All(ctx)
//
// # Collection Interface
//
// The store accepts any value satisfying the [Collection] interface, which
// is implemented by *mongo.Collection and can be mocked for testing. The
// interface requires InsertOne, Find, and DeleteMany methods.
//
// # Storage Format
//
// Each message is stored as a BSON document with fields for sequence number,
// role, parts, metadata, tool calls, and timestamp. Documents are sorted by
// the sequence field for chronological retrieval.
package mongodb
