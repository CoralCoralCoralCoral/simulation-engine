package indexer

// Update represents a location or state update
type Update struct {
	SimId               string       `json:"sim_id"`
	AgentId             string       `json:"agent_id"`
	Epoch               int64        `json:"epoch"`
	UpdateType          string       `json:"update_type"`
	State               string       `json:"state"`
	LocationId          string       `json:"location_id"`
	LocationCoordinates *Coordinates `json:"location_coordinates"`
}

// Location represents a geo-coordinate
type Coordinates struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
}
