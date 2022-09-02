# go-mongo-apicursor

A golang implementation of the Relay GraphQL Cursor Connections
Specification (https://relay.dev/graphql/connections.htm)
for MongoDB. Apache License. Will also work with REST apis to solve the same problem of sharing a cursor to MongoDB
query results over a web service request/response.

### Prerequisites

If you want to run the linter you will need golangci installed.

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
	Edges []*PersonEdge
	// A list of nodes.
	Nodes []*Person
	// Information to aid in pagination.
	PageInfo *apicursor.PageInfo
	// Identifies the total count of items in the connection.
	TotalCount uint64
}

func (dc *PersonConnection) GetEdges() []*PersonEdge {
	return dc.Edges
}

func (dc *PersonConnection) SetEdges(edges []apicursor.Edge) {
	dc.Edges = make([]*PersonEdge, len(edges))
	for i, d := range edges {
		de := d.(*PersonEdge)
		dc.Edges[i] = de
	}
}

func (dc *PersonConnection) GetNodes() []*Person {
	return dc.Nodes
}

func (dc *PersonConnection) SetNodes(nodes []interface{}) {
	dc.Nodes = make([]*Person, len(nodes))
	for i, d := range nodes {
		person := d.(*Person)
		dc.Nodes[i] = person
	}
}

func (dc *PersonConnection) GetPageInfo() *apicursor.PageInfo {
	return dc.PageInfo
}

func (dc *PersonConnection) SetPageInfo(pginfo *apicursor.PageInfo) {
	dc.PageInfo = pginfo
}

func (dc *PersonConnection) GetTotalCount() uint64 {
	return dc.TotalCount
}

func (dc *PersonConnection) SetTotalCount(count uint64) {
	dc.TotalCount = count
}

type PersonEdge struct {
	// A cursor for use in pagination.
	Cursor string
	// The item at the end of the edge.
	Node *Person
}

func (de *PersonEdge) GetCursor() string {
	return de.Cursor
}

func (de *PersonEdge) SetCursor(c string) {
	de.Cursor = c
}

func (de *PersonEdge) GetNode() interface{} {
	return de.Node
}

func (de *PersonEdge) SetNode(node interface{}) {
	de.Node = node.(*Person)
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
	return c.AddCursorFilters(findFilter, naturalSortDirection,
		apicursor.CursorFilterField{
			FieldName: "createdTime",
			FieldType: apicursor.CursorFieldTypeTime,
		},
		apicursor.CursorFilterField{
			FieldName: "_id",
			FieldType: apicursor.CursorFieldTypeMongoOid,
		},
	)
}

func (dcm PersonCursorMarshaler) Marshal(obj interface{}) (cursorFields map[string]string, err error) {
	person := obj.(*Person)
	cursorFields = make(map[string]string)
	cursorFields["createdTime"], err = apicursor.MarshalTimeField(person.CreatedTime)
	cursorFields["_id"] = apicursor.MarshalMongoOidField(person.Id)
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