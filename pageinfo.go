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
