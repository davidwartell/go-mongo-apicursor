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
	"context"
	"encoding/base64"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"strings"
	"time"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const MaxLimitAllowed = int32(1000)
const DefaultLimit = int32(10)

const (
	GreaterThanFilterOperator = "$gt"
	LessThanFilterOperator    = "$lt"
)

type APICursor interface {
	// LoadFromAPIRequest loads a cursor from query input and sets the modelFactory.
	LoadFromAPIRequest(after *string, before *string, first *int, last *int, modelFactory ModelFactory) (err error)

	// SetTimeCursorFilter attempts to apply fieldName to the filter parsed as a time.Time
	SetTimeCursorFilter(findFilter bson.M, fieldName string, naturalSortDirection int) (err error)

	// SetUUIDCursorFilter attempts to apply fieldName to the filter parsed as a mongouuid.UUID
	SetUUIDCursorFilter(findFilter bson.M, fieldName string, naturalSortDirection int) (err error)

	// SetStringCursorFilter attempts to apply fieldName to the filter parsed as a string
	SetStringCursorFilter(findFilter bson.M, fieldName string, naturalSortDirection int) (err error)

	// FindLimit calculates the limit to a database query based on requested count
	FindLimit() int64

	// CursorFilterSortDirection returns the sort direction (1 if ascending) or (-1 if descending)
	CursorFilterSortDirection(naturalSortDirection int) int

	// UnmarshalMongo unmarshals the cursor and applies it to a mongo map for use in a find filter.
	UnmarshalMongo(findFilter bson.M, naturalSortDirection int) (err error)

	// SetMarshaler sets the CursorMarshaler to use.  This must be set before calling UnmarshalMongo or MarshalMongo, or they will return err.
	SetMarshaler(cursorMarshaler CursorMarshaler)

	// ConnectionFromMongoCursor takes a mongo cursor and returns a connection.  SetMarshaler must be called first.
	ConnectionFromMongoCursor(ctx context.Context, mongoCursor *mongo.Cursor, totalDocsMatching int64) (connection Connection, err error)

	// SetModelFactory sets the modelFactory.
	SetModelFactory(modelFactory ModelFactory)

	// AfterCursor marshals the after cursor to a string.
	AfterCursor() (cursr string, err error)

	// BeforeCursor marshals the after cursor to a string.
	BeforeCursor() (cursr string, err error)
}

// ModelFactory is used to construct new objects when loading results
type ModelFactory interface {
	// New returns a new instance of the object
	New() interface{}

	NewConnection() Connection

	NewEdge() Edge
}

// CursorMarshaler is used to marshal and unmarshal cursors to and from data store query filters.
type CursorMarshaler interface {
	// UnmarshalMongo attempts to apply cursors to a mongo filter map if they exist.  Returns an error if the cursor is invalid.
	// c is the cursor to Unmarshal by calling SetTYPECursorFilter() functions. findFilter is the mongo find filter map to apply the cursor to.
	UnmarshalMongo(c APICursor, findFilter bson.M, naturalSortDirection int) (err error)

	// Marshal accepts a model type and returns an error or a map of key values for the document fields to serialize in the cursor.
	Marshal(obj interface{}) (cursorFields map[string]string, err error)
}

type DocumentCursorText func(document interface{}) (err error)

type request struct {
	after  *string
	before *string
	first  *int32
	last   *int32
}

type cursor struct {
	requestParams   request
	after           map[string]string
	before          map[string]string
	modelFactory    ModelFactory
	cursorMarshaler CursorMarshaler
}

//goland:noinspection GoUnusedExportedFunction
func NewCursor() (c APICursor) {
	c = newCursor()
	return
}

func newCursor() (c *cursor) {
	return &cursor{}
}

func (c *cursor) AfterCursor() (cursr string, err error) {
	if len(c.after) == 0 {
		cursr = ""
		return
	}
	cursr, err = json.MarshalToString(c.after)
	if err != nil {
		return
	}
	cursr = base64.URLEncoding.EncodeToString([]byte(cursr))
	return
}

func (c *cursor) BeforeCursor() (cursr string, err error) {
	if len(c.before) == 0 {
		cursr = ""
		return
	}
	cursr, err = json.MarshalToString(c.before)
	if err != nil {
		return
	}
	cursr = base64.URLEncoding.EncodeToString([]byte(cursr))
	return
}

func (c *cursor) SetModelFactory(modelFactory ModelFactory) {
	c.modelFactory = modelFactory
}

func (c *cursor) UnmarshalMongo(findFilter bson.M, naturalSortDirection int) (err error) {
	if c.cursorMarshaler == nil {
		err = errors.New("cursor marshaler is nil")
		return
	}
	err = c.cursorMarshaler.UnmarshalMongo(c, findFilter, naturalSortDirection)
	return
}

func (c *cursor) SetMarshaler(cursorMarshaler CursorMarshaler) {
	c.cursorMarshaler = cursorMarshaler
}

func (c *cursor) ConnectionFromMongoCursor(ctx context.Context, mongoCursor *mongo.Cursor, totalDocsMatching int64) (connection Connection, err error) {
	if c.modelFactory == nil {
		err = errors.New("model factory is nil")
		return
	}
	if c.cursorMarshaler == nil {
		err = errors.New("cursor marshaler is nil")
		return
	}

	connection = c.modelFactory.NewConnection()
	connection.SetTotalCount(uint64(totalDocsMatching))
	var edges []Edge
	var nodes []interface{}

	// for each result on the mongo cursor create a node and Edge
	for mongoCursor.Next(ctx) {
		newNode := c.modelFactory.New()
		newEdge := c.modelFactory.NewEdge()

		err = mongoCursor.Decode(newNode)
		if err != nil {
			err = errors.Wrapf(err, "error on Decode for cursor.ConnectionFromMongoCursor: %v", err)
			return
		}
		newEdge.SetNode(newNode)

		if c.isForward() {
			nodes = append(nodes, newNode)
			edges = append(edges, newEdge)
		} else {
			nodes = append(nodes, newNode)
			copy(nodes[1:], nodes)
			nodes[0] = newNode

			edges = append(edges, newEdge)
			copy(edges[1:], edges)
			edges[0] = newEdge
		}

		// keep going until we exhaust cursor or until we get 1 more than limit
		if int32(len(nodes)) > c.limit() {
			break
		}
	}
	if mongoCursor.Err() != nil {
		err = errors.Wrapf(mongoCursor.Err(), "error on cursor.Next for cursor.LoadResults: %v", mongoCursor.Err())
		return
	}

	// decorate all the Edges with a cursor which is the cursor to supply to an after argument to start a page at this Edge.
	newEdgeCursor := newCursor()

	// add cursors to all the edges
	for i := 0; i < len(edges); i++ {
		newEdgeCursor = newCursor()
		var cursorFields map[string]string
		cursorFields, err = c.cursorMarshaler.Marshal(edges[i].GetNode())
		if err != nil {
			return
		}
		newEdgeCursor.after = cursorFields
		var edgeCursrStr string
		edgeCursrStr, err = newEdgeCursor.AfterCursor()
		if err != nil {
			return
		}
		edges[i].SetCursor(edgeCursrStr)
	}

	pageInfo := &PageInfo{}
	if !c.isForward() {
		// if paging backward see if we have more
		if c.isBefore() {
			pageInfo.HasNextPage = true
		}
		if nodes != nil && int32(len(nodes)) > c.limit() {
			pageInfo.HasPreviousPage = true
			// trim the first element if we returned more
			nodes = nodes[1:]
			edges = edges[1:]
		}
	} else {
		// if paging forward see if we have more
		if c.isAfter() {
			pageInfo.HasPreviousPage = true
		}
		if nodes != nil && int32(len(nodes)) > c.limit() {
			pageInfo.HasNextPage = true
			// trim the last element if we returned more
			nodes = nodes[:len(nodes)-1]
			edges = edges[:len(edges)-1]
		}
	}

	connection.SetNodes(nodes)
	connection.SetEdges(edges)
	connection.SetPageInfo(pageInfo)

	if len(edges) > 0 {
		//set EndCursor
		var endCursorFields map[string]string
		var endCursorStr string
		newAfterCursor := newCursor()
		endCursorFields, err = c.cursorMarshaler.Marshal(edges[len(edges)-1].GetNode())
		if err != nil {
			return
		}
		newAfterCursor.after = endCursorFields
		endCursorStr, err = newAfterCursor.AfterCursor()
		if err != nil {
			return
		}
		pageInfo.EndCursor = &endCursorStr

		//set StartCursor
		var startCursorFields map[string]string
		var startCursorStr string
		newBeforeCursor := newCursor()
		startCursorFields, err = c.cursorMarshaler.Marshal(edges[0].GetNode())
		if err != nil {
			return
		}
		newBeforeCursor.before = startCursorFields
		startCursorStr, err = newBeforeCursor.BeforeCursor()
		if err != nil {
			return
		}
		pageInfo.StartCursor = &startCursorStr
	}

	return
}

// LoadFromAPIRequest is used to load the cursor to be used in a query request to the API layer.
func (c *cursor) LoadFromAPIRequest(after *string, before *string, first *int, last *int, modelFactory ModelFactory) (err error) {
	c.cursorMarshaler = nil
	c.modelFactory = modelFactory
	c.requestParams.after = nil
	c.requestParams.before = nil
	c.requestParams.first = nil
	c.requestParams.last = nil
	c.after = nil
	c.before = nil

	// if both first and last are passed we ignore last
	if first != nil {
		newFirst := scrubLimit(*first)
		c.requestParams.first = &newFirst
	} else if last != nil {
		newLast := scrubLimit(*last)
		c.requestParams.last = &newLast
	}

	// if neither are passed set after to default
	if c.requestParams.first == nil && c.requestParams.last == nil {
		newFirst := scrubLimit(-1)
		c.requestParams.first = &newFirst
	}

	// only use after if first is specified
	if c.isFirst() && after != nil && len(*after) > 0 {
		var cursorBytes []byte
		cursorBytes, err = base64.URLEncoding.DecodeString(*after)
		if err != nil {
			return
		}
		cursorStr := string(cursorBytes)

		c.after = make(map[string]string)

		err = json.UnmarshalFromString(cursorStr, &c.after)
		if err != nil {
			return
		}

		c.requestParams.after = &cursorStr
		// only use before if last is specified
	} else if c.isLast() && before != nil && len(*before) > 0 {
		var cursorBytes []byte
		cursorBytes, err = base64.URLEncoding.DecodeString(*before)
		if err != nil {
			return err
		}
		cursorStr := string(cursorBytes)

		c.before = make(map[string]string)

		err = json.UnmarshalFromString(cursorStr, &c.before)
		if err != nil {
			return
		}

		c.requestParams.before = &cursorStr
	}
	return
}

func (c *cursor) FindLimit() int64 {
	return int64(c.limit() + 1)
}

func (c *cursor) CursorFilterSortDirection(naturalSortDirection int) int {
	if c.isForward() {
		return naturalSortDirection
	}
	return naturalSortDirection * -1
}

type CursorFieldType int

const (
	CursorFieldTypeTime CursorFieldType = iota
	CursorFieldTypeMongoOid
	CursorFieldTypeString
)

func (t CursorFieldType) Int() int {
	return int(t)
}

type CursorFilterField struct {
	FieldName string
	FieldType CursorFieldType
}

func (c *cursor) AddCursorFilters(findFilter bson.M, naturalSortDirection int, filterFields ...CursorFilterField) (err error) {
	cursorValues := c.cursorFilter()
	if len(cursorValues) == 0 {
		// if no cursor specified do nothing and that's ok
		return
	}
	// Note: If we have a cursor it should be a valid one so error after here

	var filters []bson.M
	for i, filterField := range filterFields {
		var newFilterFieldValue interface{}
		newFilterFieldValue, err = c.UnmarshalFieldValue(filterField)
		if err != nil {
			return
		}

		newFilter := bson.M{
			filterField.FieldName: bson.D{{c.cursorFilterOperator(naturalSortDirection), newFilterFieldValue}},
		}
		if i == 0 {
			filters = append(filters, newFilter)
			continue
		}

		// multiple filters add the fields before this field to match $eq
		var andList bson.A
		for j := 0; j < i; j++ {
			var newPrependFilterFieldValue interface{}
			newPrependFilterFieldValue, err = c.UnmarshalFieldValue(filterFields[j])
			if err != nil {
				return
			}
			prependEqualFilter := bson.M{
				filterFields[j].FieldName: newPrependFilterFieldValue,
			}
			andList = append(andList, prependEqualFilter)
		}

		// finally add the filter
		andList = append(andList, newFilter)
		andFilter := bson.M{
			"$and": andList,
		}
		filters = append(filters, andFilter)
	}

	if len(filters) == 1 {
		for key, val := range filters[0] {
			findFilter[key] = val
		}
	} else {
		filterArray := make(bson.A, len(filters))
		for j, filter := range filters {
			filterArray[j] = filter
		}
		findFilter["$or"] = filterArray
	}

	return
}

func (c *cursor) UnmarshalFieldValue(filterField CursorFilterField) (result interface{}, err error) {
	switch filterField.FieldType {
	case CursorFieldTypeTime:
		return c.UnmarshalTimeField(filterField.FieldName)
	case CursorFieldTypeMongoOid:
		return c.UnmarshalMongoOidField(filterField.FieldName)
	case CursorFieldTypeString:
		return c.UnmarshalStringField(filterField.FieldName)
	default:
		err = errors.Errorf("Error cursor field (%s) with type (%d) not a recognized type", filterField.FieldName, filterField.FieldType.Int())
		return
	}
}

func (c *cursor) UnmarshalTimeField(fieldName string) (timeResult time.Time, err error) {
	if fieldValue, ok := c.cursorFilter()[fieldName]; ok {
		err = timeResult.UnmarshalText([]byte(fieldValue))
		if err != nil {
			err = errors.Errorf("Error cursor field (%s) invalid expecting a type (%s)", fieldName, "time")
			return
		}
		return
	} else {
		err = errors.Errorf("Error cursor field (%s) not found", fieldName)
		return
	}
}

// SetTimeCursorFilter looks for a field named fieldName in the cursor and assumes it is a time.Time
// Deprecated: use AddCursorFilters
func (c *cursor) SetTimeCursorFilter(findFilter bson.M, fieldName string, naturalSortDirection int) (err error) {
	cursorValues := c.cursorFilter()
	if len(cursorValues) == 0 {
		// if no cursor specified do nothing and that's ok
		return
	}
	// Note: If we have a cursor it should be a valid one so error after here

	var timeValue time.Time
	timeValue, err = c.UnmarshalTimeField(fieldName)
	if err != nil {
		return
	}

	findFilter[fieldName] = bson.D{{c.cursorFilterOperator(naturalSortDirection), timeValue}}
	return
}

func (c *cursor) UnmarshalMongoOidField(fieldName string) (uuid primitive.ObjectID, err error) {
	if fieldValue, ok := c.cursorFilter()[fieldName]; ok {
		uuid, err = uuidFromString(fieldValue)
		if err != nil {
			err = errors.Wrapf(err, "cursor invalid: expected field name %s to be type bson objectId", fieldName)
			return
		}
		return
	} else {
		err = errors.Errorf("Error cursor field (%s) not found", fieldName)
		return
	}
}

// SetUUIDCursorFilter looks for a field named fieldName in the cursor and assumes it is a mongouuid.UUID
// Deprecated: use AddCursorFilters
func (c *cursor) SetUUIDCursorFilter(findFilter bson.M, fieldName string, naturalSortDirection int) (err error) {
	cursorValues := c.cursorFilter()
	if len(cursorValues) == 0 {
		// if no cursor specified do nothing and that's ok
		return
	}
	// Note: If we have a cursor it should be a valid one so error after here
	var uuid primitive.ObjectID
	uuid, err = c.UnmarshalMongoOidField(fieldName)
	if err != nil {
		return
	}

	findFilter[fieldName] = bson.D{{c.cursorFilterOperator(naturalSortDirection), uuid}}
	return
}

func (c *cursor) UnmarshalStringField(fieldName string) (fieldValue string, err error) {
	var ok bool
	if fieldValue, ok = c.cursorFilter()[fieldName]; ok {
		return
	} else {
		err = errors.Errorf("Error cursor field (%s) not found", fieldName)
		return
	}
}

// SetStringCursorFilter looks for a field named fieldName in the cursor and assumes it is a string
// Deprecated: use AddCursorFilters
func (c *cursor) SetStringCursorFilter(findFilter bson.M, fieldName string, naturalSortDirection int) (err error) {
	cursorValues := c.cursorFilter()
	if len(cursorValues) == 0 {
		// if no cursor specified do nothing and that's ok
		return
	}
	var fieldValue string
	fieldValue, err = c.UnmarshalStringField(fieldName)
	if err != nil {
		return
	}

	findFilter[fieldName] = bson.D{{c.cursorFilterOperator(naturalSortDirection), fieldValue}}
	return
}

func (c *cursor) cursorFilterOperator(naturalSortDirection int) string {
	direction := c.CursorFilterSortDirection(naturalSortDirection)
	if direction > 0 {
		return GreaterThanFilterOperator
	} else {
		return LessThanFilterOperator
	}
}

func (c *cursor) isAfter() bool {
	return len(c.after) > 0
}

func (c *cursor) isBefore() bool {
	return len(c.before) > 0
}

func (c *cursor) isFirst() bool {
	return c.requestParams.first != nil && *c.requestParams.first > 0
}

func (c *cursor) isLast() bool {
	return c.requestParams.last != nil && *c.requestParams.last > 0
}

func (c *cursor) isForward() bool {
	if c.isAfter() || c.isFirst() {
		return true
	}
	if c.isBefore() || c.isLast() {
		return false
	}
	return true
}

func (c *cursor) cursorFilter() map[string]string {
	if c.isForward() {
		return c.after
	} else {
		return c.before
	}
}

func scrubLimit(newLimit int) (scrubbed int32) {
	scrubbed = int32(newLimit)
	if int32(newLimit) <= 0 {
		scrubbed = DefaultLimit
	} else if int32(newLimit) > MaxLimitAllowed {
		scrubbed = MaxLimitAllowed
	}
	return
}

func (c *cursor) limit() int32 {
	if c.isFirst() {
		return *c.requestParams.first
	} else if c.isLast() {
		return *c.requestParams.last
	}
	return DefaultLimit
}

func uuidFromString(s string) (primitive.ObjectID, error) {
	// remove any quotes
	s = strings.Replace(s, "\"", "", -1)
	o, err := primitive.ObjectIDFromHex(s)
	if err != nil {
		return primitive.NewObjectID(), errors.Wrapf(err, "String (%v) is not a valid UUID: %v", s, err)
	}
	return o, nil
}
