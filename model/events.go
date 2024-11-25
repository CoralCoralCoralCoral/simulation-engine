package model

import (
	"time"

	"github.com/CoralCoralCoralCoral/simulation-engine/logger"
	"github.com/google/uuid"
)

const EpochEnd logger.EventType = "epoch_end"
const CommandProcessed logger.EventType = "command_processed"
const AgentStateUpdate logger.EventType = "agent_state_update"
const AgentLocationUpdate logger.EventType = "agent_location_update"
const SpaceOccupancyUpdate logger.EventType = "space_occupancy_update"

type EpochEndPayload struct {
	Epoch    int64     `json:"epoch"`
	TimeStep int64     `json:"time_step"`
	Time     time.Time `json:"time"`
}

type CommandProcessedPayload struct {
	Epoch   int64   `json:"epoch"`
	Command Command `json:"command"`
}

type AgentStateUpdatePayload struct {
	Epoch int64      `json:"epoch"`
	Id    uuid.UUID  `json:"id"`
	State AgentState `json:"state"`
}

type AgentLocationUpdatePayload struct {
	Epoch      int64     `json:"epoch"`
	Id         uuid.UUID `json:"id"`
	LocationId uuid.UUID `json:"location_id"`
}

type SpaceOccupancyUpdatePayload struct {
	Epoch     int64     `json:"epoch"`
	Id        uuid.UUID `json:"id"`
	Occupants []struct {
		Id    uuid.UUID  `json:"id"`
		State AgentState `json:"state"`
	} `json:"occupants"`
}
