package model

import (
	"time"

	"github.com/CoralCoralCoralCoral/simulation-engine/logger"
	"github.com/google/uuid"
)

const SimulationInitialized logger.EventType = "simulation_initialized"
const EpochEnd logger.EventType = "epoch_end"
const CommandProcessed logger.EventType = "command_processed"
const AgentStateUpdate logger.EventType = "agent_state_update"
const AgentLocationUpdate logger.EventType = "agent_location_update"
const SpaceOccupancyUpdate logger.EventType = "space_occupancy_update"
const SpaceTestingUpdate logger.EventType = "space_testing_udpate"
const PolicyUpdate logger.EventType = "policy_update"

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
	Epoch               int64      `json:"epoch"`
	Id                  uuid.UUID  `json:"id"`
	State               AgentState `json:"state"`
	PreviousState       AgentState `json:"previous_state"`
	HasInfectionProfile bool       `json:"has_infection_profile"`

	// needed for metrics aggregation. not public and therefore not a json serialized field
	jurisdiction *Jurisdiction
}

type AgentLocationUpdatePayload struct {
	Epoch              int64     `json:"epoch"`
	Id                 uuid.UUID `json:"id"`
	LocationId         uuid.UUID `json:"location_id"`
	PreviousLocationId uuid.UUID `json:"previous_location_id"`

	agent *Agent
}

type SpaceOccupancyUpdatePayload struct {
	Epoch     int64     `json:"epoch"`
	Id        uuid.UUID `json:"id"`
	Occupants []struct {
		Id    uuid.UUID  `json:"id"`
		State AgentState `json:"state"`
	} `json:"occupants"`
}

type SpaceTestingUpdatePayload struct {
	Epoch     int64 `json:"epoch"`
	Positives int64 `json:"positives"`
	Negatives int64 `json:"negatives"`
	Backlog   int64 `json:"backlog"`
	Capacity  int64 `json:"capacity"`

	// needed for metrics aggregation. not public and therefore not a json serialized field
	jurisdiction *Jurisdiction
}

type PolicyUpdatePayload struct {
	JurisdictionId         string       `json:"jurisdiction_id"`
	IsMaskMandate          bool         `json:"is_mask_mandate"`
	IsLockdown             bool         `json:"is_lockdown"`
	TestStrategy           TestStrategy `json:"test_strategy"`
	TestCapacityMultiplier float64      `json:"test_capacity_multiplier"`
}

func (payload *AgentStateUpdatePayload) Jurisdiction() *Jurisdiction {
	return payload.jurisdiction
}

func (payload *SpaceTestingUpdatePayload) Jurisdiction() *Jurisdiction {
	return payload.jurisdiction
}
