package database

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var testClient *mongo.Client
var testDBName string = "test_oj_db"
var testCollectionName string = "test_users"
var testMongoURI string

func TestMain(m *testing.M) {
	// Set up test MongoDB URI
	testMongoURI = os.Getenv("TEST_MONGO_URI")
	if testMongoURI == "" {
		testMongoURI = "mongodb://localhost:27017"
	}

	// Connect to test MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(testMongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to test MongoDB: %v", err)
	}

	testClient = client

	// Run tests
	exitCode := m.Run()

	// Disconnect from test MongoDB
	disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer disconnectCancel()
	if testClient != nil {
		err = testClient.Disconnect(disconnectCtx)
		if err != nil {
			log.Printf("Failed to disconnect from test MongoDB: %v", err)
		}
	}

	os.Exit(exitCode)
}

func TestEnsureUniqueIndex(t *testing.T) {
	if testClient == nil {
		t.Fatal("Test MongoDB client not initialized")
	}

	// Use the test client and define a temporary DB variable for this test
	savedDB := DB
	DB = testClient
	defer func() {
		DB = savedDB // Restore the original DB variable
	}()

	collection := testClient.Database(testDBName).Collection(testCollectionName)

	// Clean up before the test
	_ = collection.Drop(context.Background())
	defer func() {
		// Clean up after the test
		_ = collection.Drop(context.Background())
	}()

	// Ensure unique index on a test field
	fieldKey := "testUniqueField"
	err := EnsureUniqueIndex(testDBName, testCollectionName, fieldKey)
	if err != nil {
		t.Fatalf("EnsureUniqueIndex failed: %v", err)
	}

	// Test case 1: Insert first document (should succeed)
	doc1 := bson.M{fieldKey: "value1", "otherField": "data1"}
	_, err = collection.InsertOne(context.Background(), doc1)
	if err != nil {
		t.Fatalf("Failed to insert first document: %v", err)
	}

	// Test case 2: Attempt to insert duplicate document (should fail)
	doc2 := bson.M{fieldKey: "value1", "otherField": "data2"}
	_, err = collection.InsertOne(context.Background(), doc2)

	if err == nil {
		t.Error("Expected duplicate key error, but insertion succeeded")
	} else if !mongo.IsDuplicateKeyError(err) {
		t.Errorf("Expected duplicate key error, got different error: %v", err)
	}

	// Test case 3: Insert a document with a different unique value (should succeed)
	doc3 := bson.M{fieldKey: "value2", "otherField": "data3"}
	_, err = collection.InsertOne(context.Background(), doc3)
	if err != nil {
		t.Fatalf("Failed to insert third document with different value: %v", err)
	}

	// Test case 4: Ensure calling EnsureUniqueIndex again does not cause error (index already exists)
	err = EnsureUniqueIndex(testDBName, testCollectionName, fieldKey)
	if err != nil {
		t.Fatalf("EnsureUniqueIndex failed when index already exists: %v", err)
	}

	// Test case 5: Ensure calling EnsureUniqueIndex with a different field does not cause error (index already exists)
	differentFieldKey := "differentField"
	err = EnsureUniqueIndex(testDBName, testCollectionName, differentFieldKey)
	if err != nil {
		t.Fatalf("EnsureUniqueIndex failed when index already exists: %v", err)
	}

	// Test case 6: Ensure calling EnsureUniqueIndex with a different field does not cause error (index already exists)
	err = EnsureUniqueIndex(testDBName, testCollectionName, differentFieldKey)
	if err != nil {
		t.Fatalf("EnsureUniqueIndex failed when index already exists: %v", err)
	}

}
