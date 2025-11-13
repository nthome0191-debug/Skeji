package validators

import "go.mongodb.org/mongo-driver/bson"

var BookingValidator = bson.M{
	"$jsonSchema": bson.M{
		"bsonType": "object",
		"required": []string{
			"business_id",
			"schedule_id",
			"service_label",
			"start_time",
			"end_time",
			"capacity",
			"status",
			"managed_by",
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

			"schedule_id": bson.M{
				"bsonType":  "string",
				"minLength": 24,
				"maxLength": 24,
			},

			"service_label": bson.M{
				"bsonType":  "string",
				"minLength": 2,
				"maxLength": 100,
			},

			"start_time": bson.M{
				"bsonType": "date",
			},

			"end_time": bson.M{
				"bsonType": "date",
			},

			"capacity": bson.M{
				"bsonType": "int",
				"minimum":  1,
				"maximum":  200,
			},

			"participants": bson.M{
				"bsonType": "object",
				"additionalProperties": bson.M{
					"bsonType": "string",
				},
			},

			"status": bson.M{
				"bsonType": "string",
				"enum": []string{
					"pending",
					"confirmed",
					"cancelled",
				},
			},

			"managed_by": bson.M{
				"bsonType": "object",
				"additionalProperties": bson.M{
					"bsonType": "string",
				},
			},

			"created_at": bson.M{
				"bsonType": "date",
			},
		},
	},
}
