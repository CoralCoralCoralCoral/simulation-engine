package model

import (
	"math"

	"github.com/CoralCoralCoralCoral/simulation-engine/protos/protos"
	"github.com/google/uuid"
)

const Susceptible AgentState = "susceptible"
const Infected AgentState = "infected"
const Infectious AgentState = "infectious"
const Immune AgentState = "immune"

type Agent struct {
	id                         uuid.UUID
	household                  *Space
	office                     *Space
	social_spaces              []*Space
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
		location:                   nil,
		location_change_epoch:      0,
		next_move_epoch:            0,
		state:                      Susceptible,
		state_change_epoch:         0,
		infection_profile:          nil,
		pulmonary_ventilation_rate: sampleNormal(0.36, 0.01),
		is_compliant:               is_compliant,
		mask_filtration_efficiency: math.Max(sampleNormal(0.95, 0.15), 1),
	}
}

func (agent *Agent) update(sim *Simulation) {
	agent.updateState(sim)
	agent.move(sim)
}

func (agent *Agent) updateState(sim *Simulation) {
	state_duration := float64((sim.epoch - agent.state_change_epoch) * sim.time_step)

	switch agent.state {
	case Susceptible:
		is_infected := sampleBernoulli(agent.pInfected(sim))

		if is_infected == 1 {
			agent.state = Infected
			agent.state_change_epoch = sim.epoch
			agent.infection_profile = sim.pathogen.generateInfectionProfile()
			agent.dispatchStateUpdateEvent(sim)
		}
	case Infected:
		if state_duration >= agent.infection_profile.incubation_period {
			agent.state = Infectious
			agent.state_change_epoch = sim.epoch
			agent.dispatchStateUpdateEvent(sim)
		}
	case Infectious:
		if state_duration >= agent.infection_profile.recovery_period {
			agent.state = Immune
			agent.state_change_epoch = sim.epoch
			agent.dispatchStateUpdateEvent(sim)
		}
	case Immune:
		if state_duration >= agent.infection_profile.immunity_period {
			agent.state = Susceptible
			agent.state_change_epoch = sim.epoch
			agent.infection_profile = nil
			agent.dispatchStateUpdateEvent(sim)
		}
	default:
		panic("this shouldn't be possible")
	}
}

func (agent *Agent) move(sim *Simulation) {
	if agent.next_move_epoch == 0 {
		// assumes agent is in household
		agent.next_move_epoch = sim.epoch + int64(math.Ceil(sampleNormal(12*60*60*1000, 4*60*60*1000)/float64(sim.time_step)))
	}

	if sim.epoch < agent.next_move_epoch {
		return
	}

	var next_location *Space = nil
	var next_location_duration float64 = 0

	switch agent.location.type_ {
	case Household:
		_, _, _, policy := agent.location.state()
		if policy.IsLockdown && agent.is_compliant {
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
	case Office, SocialSpace:
		next_location = agent.household
		next_location_duration = sampleNormal(12*60*60*1000, 4*60*60*1000)
	default:
		panic("this shouldn't happen")
	}

	if next_location != nil {
		// remove agent from current location
		agent.location.removeAgent(sim, agent)

		// push agent to next location
		next_location.addAgent(sim, agent)

		// set the agent's location to next location
		agent.location = next_location
		agent.location_change_epoch = sim.epoch
		agent.next_move_epoch = sim.epoch + int64(math.Ceil(next_location_duration/float64(sim.time_step)))
		agent.dispatchLocationUpdateEvent(sim)
	}
}

func (agent *Agent) infect(sim *Simulation) {
	agent.state = Infected
	agent.state_change_epoch = sim.epoch
	agent.infection_profile = sim.pathogen.generateInfectionProfile()
	agent.dispatchStateUpdateEvent(sim)
}

func (agent *Agent) dispatchStateUpdateEvent(sim *Simulation) {
	sim.logger.Log(&protos.Event{
		Type: protos.EventType_AgentStateUpdate,
		Payload: &protos.Event_AgentStateUpdate{
			AgentStateUpdate: &protos.AgentUpdatePayload{
				Epoch:       sim.epoch,
				Id:          agent.id.String(),
				State:       string(agent.state),
				LocationId:  agent.location.id.String(),
				LocationLat: agent.location.lat,
				LocationLon: agent.location.lon,
			},
		},
	})
}

func (agent *Agent) dispatchLocationUpdateEvent(sim *Simulation) {
	sim.logger.Log(&protos.Event{
		Type: protos.EventType_AgentLocationUpdate,
		Payload: &protos.Event_AgentLocationUpdate{
			AgentLocationUpdate: &protos.AgentUpdatePayload{
				Epoch:       sim.epoch,
				Id:          agent.id.String(),
				State:       string(agent.state),
				LocationId:  agent.location.id.String(),
				LocationLat: agent.location.lat,
				LocationLon: agent.location.lon,
			},
		},
	})
}

func (agent *Agent) pInfected(sim *Simulation) float64 {
	volume, _, total_infectious_doses, policy := agent.location.state()

	filtration_efficiency := 0.0
	if policy.IsMaskMandate && agent.is_compliant {
		filtration_efficiency = agent.mask_filtration_efficiency
	}

	dose_concentration := total_infectious_doses / volume

	p := 1 - math.Exp(-1*(1-filtration_efficiency)*dose_concentration*(agent.pulmonary_ventilation_rate/3600)*(float64(sim.time_step)/1000))

	return p
}
