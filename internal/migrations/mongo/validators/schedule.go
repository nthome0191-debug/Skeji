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
			"default_meeting_duration_min",
			"default_break_duration_min",
			"max_participants_per_slot",
			"time_zone",
			"created_at",
		},
		"additionalProperties": true,

		"properties": bson.M{
			"_id": bson.M{
				"bsonType": "objectId",
			},

			"business_id": bson.M{
				"bsonType":  "string",
				"minLength": 24,
				"maxLength": 24,
			},

			"name": bson.M{
				"bsonType":  "string",
				"minLength": 2,
				"maxLength": 100,
			},

			"city": bson.M{
				"bsonType":  "string",
				"minLength": 2,
				"maxLength": 100,
			},

			"address": bson.M{
				"bsonType":  "string",
				"minLength": 2,
				"maxLength": 200,
			},

			"start_of_day": bson.M{
				"bsonType": "string",
			},

			"end_of_day": bson.M{
				"bsonType": "string",
			},

			"working_days": bson.M{
				"bsonType": "array",
				"minItems": 1,
				"maxItems": 7,
				"items": bson.M{
					"bsonType":  "string",
					"minLength": 1,
				},
			},

			"default_meeting_duration_min": bson.M{
				"bsonType": "int",
				"minimum":  5,
				"maximum":  480,
			},

			"default_break_duration_min": bson.M{
				"bsonType": "int",
				"minimum":  0,
				"maximum":  480,
			},

			"max_participants_per_slot": bson.M{
				"bsonType": "int",
				"minimum":  1,
				"maximum":  200,
			},

			"exceptions": bson.M{
				"bsonType": "array",
				"maxItems": 10,
				"items": bson.M{
					"bsonType": "string",
				},
			},

			"time_zone": bson.M{
				"bsonType":  "string",
				"minLength": 1,
			},

			"created_at": bson.M{
				"bsonType": "date",
			},
		},
	},
}
