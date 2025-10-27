// Copyright (C) 2025 SAGE-X Project
//
// This file is part of sage-a2a-go.
//
// sage-a2a-go is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// sage-a2a-go is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with sage-a2a-go.  If not, see <https://www.gnu.org/licenses/>.

// Package protocol provides A2A Protocol v0.4.0 type definitions.
package protocol

import "github.com/a2aproject/a2a-go/a2a"

// ListTasksParams defines parameters for listing tasks with optional filtering criteria.
// Introduced in A2A Protocol v0.4.0.
type ListTasksParams struct {
	// ContextID filters tasks by context ID to get tasks from a specific conversation or session.
	ContextID string `json:"contextId,omitempty"`

	// Status filters tasks by their current status state.
	Status a2a.TaskState `json:"status,omitempty"`

	// PageSize is the maximum number of tasks to return. Must be between 1 and 100.
	// Defaults to 50 if not specified.
	PageSize int `json:"pageSize,omitempty"`

	// PageToken is a token for pagination. Use the NextPageToken from a previous
	// ListTasksResult response.
	PageToken string `json:"pageToken,omitempty"`

	// HistoryLength is the number of recent messages to include in each task's history.
	// Must be non-negative. Defaults to 0 if not specified.
	HistoryLength int `json:"historyLength,omitempty"`

	// LastUpdatedAfter filters tasks updated after this timestamp (milliseconds since epoch).
	// Only tasks with a last updated time greater than or equal to this value will be returned.
	LastUpdatedAfter int64 `json:"lastUpdatedAfter,omitempty"`

	// IncludeArtifacts specifies whether to include artifacts in the returned tasks.
	// Defaults to false to reduce payload size.
	IncludeArtifacts bool `json:"includeArtifacts,omitempty"`

	// Metadata is request-specific metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ListTasksResult is the result object for tasks/list method containing an array
// of tasks and pagination information.
// Introduced in A2A Protocol v0.4.0.
type ListTasksResult struct {
	// Tasks is an array of tasks matching the specified criteria.
	Tasks []*a2a.Task `json:"tasks"`

	// TotalSize is the total number of tasks available (before pagination).
	TotalSize int `json:"totalSize"`

	// PageSize is the maximum number of tasks returned in this response.
	PageSize int `json:"pageSize"`

	// NextPageToken is a token for retrieving the next page.
	// Empty string if no more results.
	NextPageToken string `json:"nextPageToken"`
}
