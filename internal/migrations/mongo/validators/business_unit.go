package validators

import "go.mongodb.org/mongo-driver/bson"

var BusinessUnitValidator = bson.M{
	"$jsonSchema": bson.M{
		"bsonType":             "object",
		"required":             []string{"name", "cities", "labels", "admin_phone", "created_at"},
		"additionalProperties": true,
		"properties": bson.M{
			"_id": bson.M{
				"bsonType": "objectId",
			},

			"name": bson.M{
				"bsonType":  "string",
				"minLength": 2,
				"maxLength": 100,
			},

			"cities": bson.M{
				"bsonType": "array",
				"minItems": 1,
				"maxItems": 50,
				"items": bson.M{
					"bsonType":  "string",
					"minLength": 1,
				},
			},

			"labels": bson.M{
				"bsonType": "array",
				"minItems": 1,
				"maxItems": 10,
				"items": bson.M{
					"bsonType":  "string",
					"minLength": 1,
				},
			},

			"admin_phone": bson.M{
				"bsonType":  "string",
				"minLength": 8,
				"maxLength": 16,
			},

			"maintainers": bson.M{
				"bsonType": "object",
				"additionalProperties": bson.M{
					"bsonType": "string",
				},
			},

			"priority": bson.M{
				"bsonType": "long",
				"minimum":  0,
			},

			"time_zone": bson.M{
				"bsonType": "string",
			},

			"website_urls": bson.M{
				"bsonType": "array",
				"maxItems": 5,
				"items": bson.M{
					"bsonType": "string",
				},
			},

			"created_at": bson.M{
				"bsonType": "date",
			},

			"city_label_pairs": bson.M{
				"bsonType": "array",
				"items": bson.M{
					"bsonType": "string",
				},
			},
		},
	},
}
