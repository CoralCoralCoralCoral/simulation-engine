package model

import "github.com/google/uuid"

const Quit CommandType = "quit"
const Pause CommandType = "pause"
const Resume CommandType = "resume"
const ApplyJurisdictionPolicy CommandType = "apply_jurisdiction_policy"
const ApplySpacePolicy CommandType = "apply_space_policy"

type Command struct {
	Type    CommandType `json:"type"`
	Payload interface{} `json:"payload"`
}

type CommandType string

type ApplyJurisdictionPolicyPayload struct {
	JurisdictionId uuid.UUID `json:"jurisdiction_id"`
	Policy         Policy    `json:"policy"`
}

type ApplySpacePolicyPayload struct {
	SpaceId uuid.UUID `json:"space_id"`
	Policy  Policy    `json:"policy"`
}
