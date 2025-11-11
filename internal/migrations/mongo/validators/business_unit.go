package validators

import "go.mongodb.org/mongo-driver/bson"

var BusinessUnitValidator = bson.M{
	"$jsonSchema": bson.M{
		"bsonType":             "object",
		"required":             []string{"name", "admin_phone", "created_at"},
		"additionalProperties": true,
		"properties": bson.M{
			"_id":         bson.M{"bsonType": "objectId"},
			"name":        bson.M{"bsonType": "string"},
			"cities":      bson.M{"bsonType": "array"},
			"labels":      bson.M{"bsonType": "array"},
			"admin_phone": bson.M{"bsonType": "string"},
			"maintainers": bson.M{"bsonType": "array"},
			"priority":    bson.M{"bsonType": "long"},
			"time_zone":   bson.M{"bsonType": "string"},
			"website_urls": bson.M{
				"bsonType": "array",
				"maxItems": 5,
				"items":    bson.M{"bsonType": "string"},
			},
			"created_at":  bson.M{"bsonType": "date"},
		},
	},
}
