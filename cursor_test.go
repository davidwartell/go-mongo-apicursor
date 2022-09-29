/*
 * Copyright (c) 2022 by David Wartell. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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
