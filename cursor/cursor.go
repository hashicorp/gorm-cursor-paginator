// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MIT

package cursor

// Cursor cursor data
type Cursor struct {
	After  *string `json:"after" query:"after"`
	Before *string `json:"before" query:"before"`
}
