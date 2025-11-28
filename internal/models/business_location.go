package models

// BusinessAggregatedMarker represents a single marker aggregating all active business workers.
type BusinessAggregatedMarker struct {
	BusinessUserID int      `json:"business_user_id"`
	Latitude       *float64 `json:"latitude,omitempty"`
	Longitude      *float64 `json:"longitude,omitempty"`
	WorkerCount    int      `json:"worker_count"`
}

// BusinessWorkerLocation bundles a worker payload with owning business id for WebSocket updates.
type BusinessWorkerLocation struct {
	BusinessUserID int                   `json:"business_user_id"`
	Worker         ExecutorLocationGroup `json:"worker"`
}
