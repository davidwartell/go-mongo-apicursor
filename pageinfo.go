///
// Copyright (c) 2022. StealthMode Inc. All Rights Reserved
///

package go_mongo_apicursor

type PageInfo struct {
	// When paginating forwards, the cursor to continue.
	EndCursor *string `json:"endCursor"`

	// When paginating forwards, are there more items?
	HasNextPage bool `json:"hasNextPage"`

	// When paginating backwards, are there more items?
	HasPreviousPage bool `json:"hasPreviousPage"`

	// When paginating backwards, the cursor to continue.
	StartCursor *string `json:"startCursor"`
}

// Connection implements paging according to spec: https://relay.dev/graphql/connections.htm
type Connection interface {
	//// Edges returns a list of Edges on the current page.
	//Edges() []Edge

	// SetEdges sets the edges on the current page.
	SetEdges(edges []Edge)

	//// Nodes returns a list of nodes on the current page.
	//Nodes() []interface{}

	// SetNodes sets the nodes for the current page.
	SetNodes(nodes []interface{})

	// PageInfo returns the PageInfo
	PageInfo() *PageInfo

	// SetPageInfo sets the PageInfo
	SetPageInfo(pginfo *PageInfo)

	// TotalCount returns the total count of nodes in the connection (count of all nodes matching the query in all pages).
	TotalCount() uint64

	// SetTotalCount sets the TotalCount.
	SetTotalCount(count uint64)
}

type Edge interface {
	// Cursor returns the after cursor to load cursor starting at this Edge.
	Cursor() string

	// SetCursor sets the cursor
	SetCursor(c string)

	// GetNode returns the Node at this Edge.
	GetNode() interface{}

	// SetNode sets the Node for this Edge.
	SetNode(node interface{})
}

type OrderDirection int

//goland:noinspection ALL
const (
	ASC  OrderDirection = 1
	DESC OrderDirection = -1
)

func (od OrderDirection) Int() int {
	return int(od)
}
