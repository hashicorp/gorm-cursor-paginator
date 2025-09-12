// Copyright IBM Corp. 2018, 2025
// SPDX-License-Identifier: MIT

package cursor

import "errors"

// Errors for encoder
var (
	ErrInvalidCursor = errors.New("invalid cursor")
	ErrInvalidModel  = errors.New("invalid model")
)
