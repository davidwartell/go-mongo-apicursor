package apicursor

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
	"time"
)

func TestAddCursorFiltersMulti(t *testing.T) {
	createdTime := time.Now().UTC()
	createdTimeText, err := createdTime.MarshalText()
	require.Nil(t, err)
	mongoOid := primitive.NewObjectID()

	testCursor := cursor{
		after: map[string]string{
			"created": string(createdTimeText),
			"oid":     mongoOid.Hex(),
		},
	}

	findFilter := make(bson.M)
	err = testCursor.AddCursorFilters(findFilter, ASC.Int(),
		CursorFilterField{
			FieldName: "created",
			FieldType: CursorFieldTypeTime,
		},
		CursorFilterField{
			FieldName: "oid",
			FieldType: CursorFieldTypeMongoOid,
		},
	)
	require.Nil(t, err)

	expectedFilter := primitive.M{
		"$or": primitive.A{
			primitive.M{
				"created": primitive.M{
					"$gt": createdTime,
				},
			},
			primitive.M{
				"created": createdTime,
				"oid": primitive.M{
					"$gt": mongoOid,
				},
			},
		},
	}

	assert.Equal(t, expectedFilter, findFilter)

	var jsonDebug []byte
	jsonDebug, err = json.MarshalIndent(findFilter, "", "    ")
	require.Nil(t, err)
	t.Log(string(jsonDebug))
}

func TestAddCursorFiltersSingle(t *testing.T) {
	createdTime := time.Now().UTC()
	createdTimeText, err := createdTime.MarshalText()
	require.Nil(t, err)

	testCursor := cursor{
		after: map[string]string{
			"created": string(createdTimeText),
		},
	}

	findFilter := make(bson.M)
	err = testCursor.AddCursorFilters(findFilter, ASC.Int(),
		CursorFilterField{
			FieldName: "created",
			FieldType: CursorFieldTypeTime,
		},
	)
	require.Nil(t, err)

	expectedFilter := primitive.M{
		"created": primitive.M{
			"$gt": createdTime,
		},
	}

	assert.Equal(t, expectedFilter, findFilter)

	var jsonDebug []byte
	jsonDebug, err = json.MarshalIndent(findFilter, "", "    ")
	require.Nil(t, err)
	t.Log(string(jsonDebug))
}
