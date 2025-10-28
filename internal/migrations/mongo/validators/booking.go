package validators

import "go.mongodb.org/mongo-driver/bson"

var BookingValidator = bson.M{
	"$jsonSchema": bson.M{
		"bsonType":             "object",
		"required":             []string{"business_id", "schedule_id", "start_time", "end_time", "status", "created_at"},
		"additionalProperties": true,
		"properties": bson.M{
			"business_id":   bson.M{"bsonType": "string"},
			"schedule_id":   bson.M{"bsonType": "string"},
			"service_label": bson.M{"bsonType": "string"},
			"start_time":    bson.M{"bsonType": "date"},
			"end_time":      bson.M{"bsonType": "date"},
			"capacity":      bson.M{"bsonType": "long"},
			"participants":  bson.M{"bsonType": "array"},
			"status":        bson.M{"bsonType": "string"},
			"managed_by":    bson.M{"bsonType": "string"},
			"created_at":    bson.M{"bsonType": "date"},
		},
	},
}
