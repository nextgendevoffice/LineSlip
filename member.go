package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MemberSystem struct {
	db *mongo.Database
}

func NewMemberSystem() *MemberSystem {
	return &MemberSystem{
		db: initDB(),
	}
}

func (ms *MemberSystem) AddMember(userID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := ms.db.Collection("members")
	filter := bson.M{"user_id": userID}
	update := bson.M{"$set": bson.M{"user_id": userID}}

	upsert := true
	opts := options.Update().SetUpsert(upsert)
	res, err := collection.UpdateOne(ctx, filter, update, opts)

	if err != nil {
		log.Println("Error updating member:", err)
		return
	}

	if res.UpsertedCount > 0 {
		log.Printf("Member %s added to the system", userID)
	} else {
		log.Printf("Member %s already exists in the system", userID)
	}
}

func (ms *MemberSystem) IsMember(userID string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := ms.db.Collection("members")
	filter := bson.M{"user_id": userID}

	var result bson.M
	err := collection.FindOne(ctx, filter).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("User %s not found in the member system", userID)
		} else {
			log.Printf("Error checking member status: %v", err)
		}
		return false
	}

	log.Printf("User %s found in the member system", userID)
	return true
}
