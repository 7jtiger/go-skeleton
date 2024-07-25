package models

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectDB(dbname, col string) (*mongo.Collection, error) {
	// config := conf.NewConfig(*configFlag)
	//cfg := config["redis-db"]

	connect := func(dataSource string) (*mongo.Client, error) {
		// if client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("dataSource")); err != nil {
		if client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://10.30.174.41:27017")); err != nil {
			return nil, err
		} else if err = client.Ping(context.Background(), nil); err != nil {
			return nil, err
		} else {
			return client, nil
		}
	}
	//cfg["redis-db"]
	//var err error
	if client, err := connect("mongodb://localhost:27017"); err != nil {
		return nil, err
	} else {
		db := client.Database(dbname)
		collectionInfo := db.Collection(col)
		// if err := db_util.CreateIndex(r.collectionInfo, db_util.IndexOpt{Field: "time", Order: 1}, db_util.IndexOpt{Field: "currency", Order: 1}); err != nil {
		// 	// db.userinfo.createIndex( { email: 1 }, { unique: true } )
		// 	// db.userinfo.createIndex( { firstName: 1, lastName: 1 }, { unique: true } )
		// 	return nil, err
		// }
		return collectionInfo, nil
	}
}

func Test_DBupdate(t *testing.T) {
	col, err := ConnectDB("go-ready", "tPerson")
	if err != nil {
		return
	}

	filter := bson.M{"uid": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJrZXkiOiJ3ZW1hZGV0cmVlIiwiZXhwIjozMDAsImlzcyI6IndlbWl4In0.EBxqmqScR08fr5KW_ojFOLzbD_42BXRBjRDK6qVGvas"}

	update := bson.M{
		"$set": bson.M{"status": 1},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if res, err := col.UpdateOne(ctx, filter, update); err != nil {
		return
	} else {
		fmt.Println("====== ", res)
	}
}
