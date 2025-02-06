// Copyright 2025 Interlynk.io
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// -------------------------------------------------------------------------

package tcontext

import "context"

type TransferMetadata struct {
	context.Context
	values map[string]interface{} // Stores key-value pairs
}

// WithValue adds a key-value pair to TransferMetadata
func (tm *TransferMetadata) WithValue(key string, value interface{}) {
	tm.values[key] = value
}

// Value retrieves a value from TransferMetadata
func (tm *TransferMetadata) Value(key string) interface{} {
	return tm.values[key]
}

// NewTransferMetadata initializes TransferMetadata
func NewTransferMetadata(ctx context.Context) *TransferMetadata {
	return &TransferMetadata{
		Context: ctx,
		values:  make(map[string]interface{}),
	}
}
