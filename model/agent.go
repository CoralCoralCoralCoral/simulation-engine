package model

import (
	"math"

	"github.com/CoralCoralCoralCoral/simulation-engine/logger"
	"github.com/google/uuid"
)

const Susceptible AgentState = "susceptible"
const Infected AgentState = "infected"
const Infectious AgentState = "infectious"
const Hospitalized AgentState = "hospitalized"
const Dead AgentState = "dead"
const Immune AgentState = "immune"

type Agent struct {
	id                         uuid.UUID
	household                  *Space
	office                     *Space
	social_spaces              []*Space
	healthcare_spaces          []*Space
	location                   *Space
	location_change_epoch      int64
	next_move_epoch            int64
	state                      AgentState
	state_change_epoch         int64
	infection_profile          *InfectionProfile
	pulmonary_ventilation_rate float64
	is_compliant               bool
	mask_filtration_efficiency float64
}

type AgentState string

func newAgent() Agent {
	is_compliant := false
	if sampleBernoulli(0.95) == 1 {
		is_compliant = true
	}

	return Agent{
		id:                         uuid.New(),
		household:                  nil,
		office:                     nil,
		social_spaces:              make([]*Space, 0),
		healthcare_spaces:          make([]*Space, 0),
		location:                   nil,
		location_change_epoch:      0,
		next_move_epoch:            0,
		state:                      Susceptible,
		state_change_epoch:         0,
		infection_profile:          nil,
		pulmonary_ventilation_rate: sampleNormal(0.36, 0.01),
		is_compliant:               is_compliant,
		mask_filtration_efficiency: math.Max(sampleNormal(0.8, 0.15), 1),
	}
}

func (agent *Agent) update(sim *Simulation) {
	if agent.state == Dead {
		return
	}

	agent.updateState(sim)
	agent.updateLocation(sim)
}

func (agent *Agent) updateState(sim *Simulation) {
	state_duration := float64((sim.epoch - agent.state_change_epoch) * sim.time_step)

	switch agent.state {
	case Susceptible:
		is_infected := sampleBernoulli(agent.pInfected(sim))

		if is_infected == 1 {
			agent.infection_profile = sim.pathogen.generateInfectionProfile()
			agent.setState(sim, Infected)
		}
	case Infected:
		if state_duration >= agent.infection_profile.incubation_period {
			agent.setState(sim, Infectious)
		}
	case Infectious:
		switch agent.infection_profile.is_hospitalized {
		case true:
			if state_duration >= agent.infection_profile.prehospitalization_period {
				agent.setState(sim, Hospitalized)
			}
		case false:
			if state_duration >= agent.infection_profile.recovery_period {
				agent.setState(sim, Immune)
			}
		}
	case Hospitalized:
		if state_duration >= agent.infection_profile.hospitalization_period {
			if agent.infection_profile.is_dead {
				agent.setState(sim, Dead)
			} else if agent.infection_profile != nil {
				agent.setState(sim, Immune)
			} else {
				agent.setState(sim, Susceptible)
			}
		}
	case Immune:
		if state_duration >= agent.infection_profile.immunity_period {
			agent.infection_profile = nil
			agent.setState(sim, Susceptible)
		}
	case Dead:
		// noop
	default:
		panic("this shouldn't be possible")
	}
}

func (agent *Agent) setState(sim *Simulation, state AgentState) {
	previous_state := agent.state

	agent.state = state
	agent.state_change_epoch = sim.epoch
	agent.dispatchStateUpdateEvent(sim, previous_state)
}

func (agent *Agent) updateLocation(sim *Simulation) {
	if agent.next_move_epoch == 0 {
		// assumes agent is in household
		agent.next_move_epoch = sim.epoch + int64(math.Ceil(sampleNormal(12*60*60*1000, 4*60*60*1000)/float64(sim.time_step)))
	}

	// in the special case where the agent state transitioned to
	// Hospitalized in this epoch, we set next_move_epoch to this epoch
	if agent.state == Hospitalized && agent.state_change_epoch == sim.epoch {
		agent.next_move_epoch = sim.epoch
	}

	if sim.epoch < agent.next_move_epoch {
		return
	}

	var next_location *Space = nil
	var next_location_duration float64 = 0

	if agent.state == Hospitalized && agent.state_change_epoch == sim.epoch {
		next_location = agent.healthcare_spaces[sampleUniform(0, int64(len(agent.healthcare_spaces)-1))]
		next_location_duration = agent.infection_profile.hospitalization_period
	} else {
		location_type, _, _, _, policy := agent.location.state()
		switch location_type {
		case Household:
			if policy != nil && policy.IsLockDown && agent.is_compliant {
				break
			}

			if sampleBernoulli(0.55) == 1 {
				next_location = agent.office
				next_location_duration = sampleNormal(8*60*60*1000, 2*60*60*1000)
			} else {
				// select a social space at uniform random from the agent's list of social spaces.
				// in reality this wouldn't be uniform random, rather mostly a function of proximity,
				// which can be implemented once geospatial attributes are added to spaces
				next_location = agent.social_spaces[sampleUniform(0, int64(len(agent.social_spaces)-1))]
				next_location_duration = sampleNormal(45*60*1000, 15*60*1000)
			}
		case Office, SocialSpace, HealthCareSpace:
			next_location = agent.household
			next_location_duration = sampleNormal(12*60*60*1000, 4*60*60*1000)
		default:
			panic("this shouldn't happen")
		}
	}

	if next_location != nil {
		agent.setLocation(sim, next_location, next_location_duration)
	}
}

func (agent *Agent) setLocation(sim *Simulation, location *Space, duration float64) {
	previous_location_id := agent.location.id

	// remove agent from current location
	agent.location.removeAgent(sim, agent)

	// push agent to next location
	location.addAgent(sim, agent)

	// set the agent's location to next location
	agent.location = location
	agent.location_change_epoch = sim.epoch
	agent.next_move_epoch = sim.epoch + int64(math.Ceil(duration/float64(sim.time_step)))
	agent.dispatchLocationUpdateEvent(sim, previous_location_id)
}

func (agent *Agent) infect(sim *Simulation) {
	agent.infection_profile = sim.pathogen.generateInfectionProfile()
	agent.setState(sim, Infected)
}

func (agent *Agent) dispatchStateUpdateEvent(sim *Simulation, previous_state AgentState) {
	event := logger.Event{
		Type: AgentStateUpdate,
		Payload: AgentStateUpdatePayload{
			Epoch:               sim.epoch,
			Id:                  agent.id,
			State:               agent.state,
			PreviousState:       previous_state,
			HasInfectionProfile: agent.infection_profile != nil,

			jurisdiction: agent.household.jurisdiction,
		},
	}

	sim.logger.Log(event)
}

func (agent *Agent) dispatchLocationUpdateEvent(sim *Simulation, previous_location_id uuid.UUID) {
	event := logger.Event{
		Type: AgentLocationUpdate,
		Payload: AgentLocationUpdatePayload{
			Epoch:              sim.epoch,
			Id:                 agent.id,
			LocationId:         agent.location.id,
			PreviousLocationId: previous_location_id,
		},
	}

	sim.logger.Log(event)
}

func (agent *Agent) pInfected(sim *Simulation) float64 {
	_, volume, _, total_infectious_doses, policy := agent.location.state()

	filtration_efficiency := 0.0
	if policy != nil && policy.IsMaskMandate && agent.is_compliant {
		filtration_efficiency = agent.mask_filtration_efficiency
	}

	dose_concentration := total_infectious_doses / volume

	p := 1 - math.Exp(-1*(1-filtration_efficiency)*dose_concentration*(agent.pulmonary_ventilation_rate/3600)*(float64(sim.time_step)/1000))

	return p
}
