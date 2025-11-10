package validators

import "go.mongodb.org/mongo-driver/bson"

var ScheduleValidator = bson.M{
	"$jsonSchema": bson.M{
		"bsonType": "object",
		"required": []string{
			"business_id",
			"name",
			"city",
			"address",
			"start_of_day",
			"end_of_day",
			"working_days",
			"time_zone",
			"created_at",
		},
		"additionalProperties": true,
		"properties": bson.M{
			"_id":                          bson.M{"bsonType": "objectId"},
			"business_id":                  bson.M{"bsonType": "string"},
			"name":                         bson.M{"bsonType": "string"},
			"city":                         bson.M{"bsonType": "string"},
			"address":                      bson.M{"bsonType": "string"},
			"start_of_day":                 bson.M{"bsonType": "string"},
			"end_of_day":                   bson.M{"bsonType": "string"},
			"working_days":                 bson.M{"bsonType": "array"},
			"default_meeting_duration_min": bson.M{"bsonType": "int"},
			"default_break_duration_min":   bson.M{"bsonType": "int"},
			"max_participants_per_slot":    bson.M{"bsonType": "int"},
			"exceptions":                   bson.M{"bsonType": "array"},
			"time_zone":                    bson.M{"bsonType": "string"},
			"created_at":                   bson.M{"bsonType": "date"},
		},
	},
}
