// Copyright 2025 Interlynk.io
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

// AdapterRole defines whether the adapter is input or output
type AdapterRole string

const (
	InputAdapterRole  AdapterRole = "input"
	OutputAdapterRole AdapterRole = "output"
)

type AdapterType string

const (
	GithubAdapterType    AdapterType = "github"
	InterlynkAdapterType AdapterType = "interlynk"
	FolderAdapterType    AdapterType = "folder"
	DtrackAdapterType    AdapterType = "dtrack"
)

type ProcessingMode string

const (
	FetchParallel   ProcessingMode = "parallel"
	FetchSequential ProcessingMode = "sequential"
)

type UploadMode string

const (
	UploadParallel   UploadMode = "parallel"
	UploadBatching   UploadMode = "batch"
	UploadSequential UploadMode = "sequential"
)

// UploadSettings contains configuration for SBOM uploads
type UploadSettings struct {
	ProcessingMode UploadMode // "sequential", "parallel", or "batch"
}
