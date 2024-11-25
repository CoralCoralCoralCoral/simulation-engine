package model

import (
	"math"
	"time"

	"github.com/CoralCoralCoralCoral/simulation-engine/logger"
	"github.com/google/uuid"
)

type Simulation struct {
	id            uuid.UUID
	pathogen      Pathogen
	start_time    time.Time
	epoch         int64
	time_step     int64
	agents        []*Agent
	jurisdictions []*Jurisdiction
	households    []*Space
	offices       []*Space
	social_spaces []*Space
	is_paused     bool
	should_quit   bool
	commands      chan Command
	logger        logger.Logger
}

func NewSimulation(config Config) Simulation {
	jurisdictions := createJurisdictions()
	households := createHouseholds(config.NumAgents)
	offices := createOffices(config.NumAgents)
	social_spaces := createSocialSpaces(config.NumAgents / 100)
	agents := createAgents(config.NumAgents, households, offices, social_spaces)

	logger := logger.NewLogger()

	return Simulation{
		id:            config.Id,
		pathogen:      config.Pathogen,
		start_time:    time.Now(),
		epoch:         0,
		time_step:     config.TimeStep,
		agents:        agents,
		jurisdictions: jurisdictions,
		households:    households,
		offices:       offices,
		social_spaces: social_spaces,
		commands:      make(chan Command),
		logger:        logger,
	}
}

func (sim *Simulation) Start() {
	go sim.logger.Broadcast()

	sim.infectRandomAgent()

	for {
		if sim.should_quit {
			return
		}

		select {
		case command := <-sim.commands:
			sim.processCommand(command)
		default:
			sim.simulateEpoch()
		}
	}
}

func (sim *Simulation) Subscribe(subscriber func(event *logger.Event)) {
	sim.logger.Subscribe(subscriber)
}

func (sim *Simulation) SendCommand(command Command) {
	sim.commands <- command
}

func (sim *Simulation) Id() uuid.UUID {
	return sim.id
}

func (sim *Simulation) processCommand(command Command) {
	switch command.Type {
	case Quit:
		sim.should_quit = true
	case Pause:
		sim.is_paused = true
	case Resume:
		sim.is_paused = false
	case ApplyJurisdictionPolicy:
		if payload, ok := command.Payload.(ApplyJurisdictionPolicyPayload); ok {
			sim.applyJurisdictionPolicy(payload)
		}
	case ApplySpacePolicy:
		if payload, ok := command.Payload.(ApplySpacePolicyPayload); ok {
			sim.applySpacePolicy(payload)
		}
	}

	sim.logger.Log(logger.Event{
		Type: CommandProcessed,
		Payload: CommandProcessedPayload{
			Epoch:   sim.epoch,
			Command: command,
		},
	})
}

func (sim *Simulation) simulateEpoch() {
	if sim.is_paused {
		return
	}

	sim.epoch = sim.epoch + 1

	for _, agent := range sim.agents {
		agent.update(sim)
	}

	for _, household := range sim.households {
		household.update(sim)
	}

	for _, office := range sim.offices {
		office.update(sim)
	}

	for _, social_space := range sim.social_spaces {
		social_space.update(sim)
	}

	sim.logger.Log(logger.Event{
		Type: EpochEnd,
		Payload: EpochEndPayload{
			Epoch:    sim.epoch,
			TimeStep: sim.time_step,
			Time:     sim.time(),
		},
	})
}

func (sim *Simulation) infectRandomAgent() {
	agent_idx := sampleUniform(0, int64(len(sim.agents)-1))
	sim.agents[agent_idx].infect(sim)
}

func (sim *Simulation) time() time.Time {
	return sim.start_time.Add(time.Duration(sim.epoch*sim.time_step) * time.Millisecond)
}

func (sim *Simulation) applyJurisdictionPolicy(payload ApplyJurisdictionPolicyPayload) {
	for _, jur := range sim.jurisdictions {
		if jur.id == payload.JurisdictionId {
			jur.applyPolicy(&payload.Policy)
			return
		}
	}
}

func (sim *Simulation) applySpacePolicy(payload ApplySpacePolicyPayload) {
	for _, space := range sim.households {
		if space.id == payload.SpaceId {
			space.applyPolicy(&payload.Policy)
			return
		}
	}

	for _, space := range sim.offices {
		if space.id == payload.SpaceId {
			space.applyPolicy(&payload.Policy)
			return
		}
	}

	for _, space := range sim.social_spaces {
		if space.id == payload.SpaceId {
			space.applyPolicy(&payload.Policy)
			return
		}
	}
}

func createJurisdictions() []*Jurisdiction {
	jurisdictions := make([]*Jurisdiction, 0)

	return jurisdictions
}

func createHouseholds(total_capacity int64) []*Space {
	households := make([]*Space, 0)

	for remaining_capacity := total_capacity; remaining_capacity > 0; {
		capacity := int64(math.Max(math.Floor(sampleNormal(4, 1)), 1))

		if capacity > remaining_capacity {
			capacity = remaining_capacity
		}

		household := newHousehold(capacity)
		households = append(households, &household)

		remaining_capacity -= capacity
	}

	return households
}

func createOffices(total_capacity int64) []*Space {
	offices := make([]*Space, 0)

	for remaining_capacity := total_capacity; remaining_capacity > 0; {
		capacity := int64(math.Max(math.Floor(sampleNormal(10, 2)), 1))

		if capacity > remaining_capacity {
			capacity = remaining_capacity
		}

		office := newOffice(capacity)
		offices = append(offices, &office)

		remaining_capacity -= capacity
	}

	return offices
}

func createSocialSpaces(total_capacity int64) []*Space {
	social_spaces := make([]*Space, 0)

	for remaining_capacity := total_capacity; remaining_capacity > 0; {
		capacity := int64(math.Max(math.Floor(sampleNormal(10, 2)), 1))

		if capacity > remaining_capacity {
			capacity = remaining_capacity
		}

		social_space := newSocialSpace(capacity)
		social_spaces = append(social_spaces, &social_space)

		remaining_capacity -= capacity
	}

	return social_spaces
}

func createAgents(count int64, households, offices []*Space, social_spaces []*Space) []*Agent {
	agents := make([]*Agent, count)

	for i := 0; i < int(count); i++ {
		agent := newAgent()

		num_social_spaces := int(math.Max(1, math.Floor(sampleNormal(5, 4))))
		for i := 0; i < num_social_spaces; i++ {
			agent.social_spaces = append(agent.social_spaces, social_spaces[sampleUniform(0, int64(len(social_spaces)-1))])
		}

		agents[i] = &agent
	}

	// allocate agents to households
	household_idx, household_allocated_capacity := 0, 0
	for _, agent := range agents {
		household := households[household_idx]
		agent.household = household
		agent.location = household

		household_allocated_capacity += 1
		if household_allocated_capacity == int(household.capacity) {
			household_idx += 1
			household_allocated_capacity = 0
		}
	}

	// allocate agents to offices
	office_idx, office_allocated_capacity := 0, 0
	for _, agent := range agents {
		office := offices[office_idx]
		agent.office = office

		office_allocated_capacity += 1
		if office_allocated_capacity == int(office.capacity) {
			office_idx += 1
			office_allocated_capacity = 0
		}
	}

	return agents
}
