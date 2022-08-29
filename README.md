# go-mongo-apicursor

Go code for helping with keeping MongoDB cursors across web service calls, especially GraphQL. Apache License.

### Prerequisites

HomeBrew: https://brew.sh/

```
brew install golangci/tap/golangci-lint
```

### Building
```
make setup
make all
```

### Building for Release
```
make setup
make release all
```

### Unit Tests
```
make test
```

* Linter

```
make golint
```

* Security Linter

```
make gosec
```

### Example
```
type Person struct {
    Id bson.ObjectId `bson:"_id"`
    Name string `bson:"name"`
    CreatedTime time.Time `bson:"createdTime"`
}

type PersonFactory struct {
}

func (df *PersonFactory) New() interface{} {
	return NewPerson()
}

func (df *PersonFactory) NewConnection() apicursor.Connection {
	doc := PersonConnection{}
	return &doc
}

func (df *PersonFactory) NewEdge() apicursor.Edge {
	doc := PersonEdge{}
	return &doc
}

type PersonConnection struct {
	// A list of edges.
	edges []*PersonEdge
	// A list of nodes.
	nodes []*Person
	// Information to aid in pagination.
	pageInfo *apicursor.PageInfo
	// Identifies the total count of items in the connection.
	totalCount uint64
}

func (dc *PersonConnection) GetEdges() []*PersonEdge {
	return dc.edges
}

func (dc *PersonConnection) SetEdges(edges []apicursor.Edge) {
	dc.edges = make([]*PersonEdge, len(edges))
	for i, d := range edges {
		de := d.(*PersonEdge)
		dc.edges[i] = de
	}
}

func (dc *PersonConnection) GetNodes() []*Person {
	return dc.nodes
}

func (dc *PersonConnection) SetNodes(nodes []interface{}) {
	dc.nodes = make([]*Person, len(nodes))
	for i, d := range nodes {
		dev := d.(*Person)
		dc.nodes[i] = dev
	}
}

func (dc *PersonConnection) GetPageInfo() *apicursor.PageInfo {
	return dc.pageInfo
}

func (dc *PersonConnection) SetPageInfo(pginfo *apicursor.PageInfo) {
	dc.pageInfo = pginfo
}

func (dc *PersonConnection) GetTotalCount() uint64 {
	return dc.totalCount
}

func (dc *PersonConnection) SetTotalCount(count uint64) {
	dc.totalCount = count
}

type PersonEdge struct {
	// A cursor for use in pagination.
	cursor string
	// The item at the end of the edge.
	node *Person
}

func (de *PersonEdge) GetCursor() string {
	return de.cursor
}

func (de *PersonEdge) SetCursor(c string) {
	de.cursor = c
}

func (de *PersonEdge) GetNode() interface{} {
	return de.node
}

func (de *PersonEdge) SetNode(node interface{}) {
	de.node = node.(*Person)
}

// PersonOrder Ordering options for Person connections
type PersonOrder struct {
	// The ordering direction.
	Direction apicursor.OrderDirection `json:"direction"`
}

func (r *queryResolver) Persons(ctx context.Context, after *string, before *string, first *int, last *int, orderBy *PersonOrder) (*PersonConnection, error) {
    cursor := apicursor.NewCursor()
	err = cursor.LoadFromAPIRequest(after, before, first, last, &PersonFactory{})
	if err != nil {
		return err
	}
	return person.Instance().Find(
		ctx,
		cursor,
		orderBy,
	)
}

type PersonCursorMarshaler struct{}

func (dcm PersonCursorMarshaler) UnmarshalMongo(c apicursor.APICursor, findFilter bson.M, naturalSortDirection int) (err error) {
    timeFieldName, idFieldName := "createdTime", "_id"
    cursorFilters := bson.M{}
	err = c.SetTimeCursorFilter(cursorFilters, "createdTime", naturalSortDirection)
	if err != nil {
		return
	}
	err = c.SetUUIDCursorFilter(cursorFilters, "_id", naturalSortDirection)
	if err != nil {
		return
	}
	if len(cursorFilters) > 0 {
		timeFilter := cursorFilters[timeFieldName].(bson.D)[0]
		idFilter := cursorFilters[idFieldName].(bson.D)[0]
		findFilter["$or"] = bson.A{
			bson.M{timeFieldName: bson.M{timeFilter.Key: timeFilter.Value}},
			bson.M{
				timeFieldName: timeFilter.Value,
				idFieldName:       bson.M{idFilter.Key: idFilter.Value},
			},
		}
	}
	return
}

func (dcm PersonCursorMarshaler) Marshal(obj interface{}) (cursorFields map[string]string, err error) {
	dev := obj.(*Person)
	cursorFields = make(map[string]string)
	var marshaledBytes []byte
	marshaledBytes, err = dev.ServerCreated.MarshalText()
	if err != nil {
		return
	}
	cursorFields["createdTime"] = string(marshaledBytes)
	cursorFields["_id"] = dev.SortCursor.String()
	return
}

func (s *personService) Find(ctx context.Context, queryCursor apicursor.APICursor, orderBy *PersonOrder) (personConnection *PersonConnection, err error) {
	if queryCursor == nil {
		queryCursor = apicursor.NewCursor()
	}
	queryCursor.SetMarshaler(&PersonCursorMarshaler{})
	
	filter := bson.M{}
	
	findOptions := &options.FindOptions{}
	findOptions.SetLimit(queryCursor.FindLimit())
	sortDirection := apicursor.DESC.Int()
	if orderBy != nil {
		sortDirection = orderBy.Direction.Int()
	}
	findOptions.SetSort(bson.D{
		{"createdTime", queryCursor.CursorFilterSortDirection(sortDirection)},
		{"_id", queryCursor.CursorFilterSortDirection(sortDirection)},
	})
	
	var countDocsResult int64
	countDocsResult, err = collection.CountDocuments(ctx, filter)
	if err != nil {
		return
	}
	
	err = queryCursor.UnmarshalMongo(filter, sortDirection)
	if err != nil {
		return
	}
	
	var mongoCursor *mongo.Cursor
	mongoCursor, err = collection.Find(ctx, filter, findOptions)
	if err != nil {
		return
	}
	defer func() {
		if mongoCursor != nil {
			_ = mongoCursor.Close(ctx)
		}
	}()

	var connectionResult apicursor.Connection
	connectionResult, err = queryCursor.ConnectionFromMongoCursor(ctx, mongoCursor, countDocsResult)
	if err != nil {
		return
	}
	personConnection = connectionResult.(*PersonConnection)
	return
}
```