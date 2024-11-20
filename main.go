package main

import (
	"log"

	"github.com/CoralCoralCoralCoral/simulation-engine/indexer"
	"github.com/CoralCoralCoralCoral/simulation-engine/messaging"
	"github.com/CoralCoralCoralCoral/simulation-engine/model"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// create a game with 150k people
	sim := model.NewSimulation(model.Config{
		Id:        uuid.New(),
		NumAgents: 150000,
		TimeStep:  15 * 60 * 1000,
		Pathogen: model.Pathogen{
			IncubationPeriod:   [2]float64{3 * 24 * 60 * 60 * 1000, 8 * 60 * 60 * 1000},
			RecoveryPeriod:     [2]float64{7 * 24 * 60 * 60 * 1000, 8 * 60 * 60 * 1000},
			ImmunityPeriod:     [2]float64{330 * 24 * 60 * 60 * 1000, 90 * 24 * 60 * 60 * 1000},
			QuantaEmissionRate: [2]float64{250, 100},
		},
	})

	// start a new metrics instance subscribed to simulation events
	sim.Subscribe(messaging.NewMetricsTransmitter(uuid.Max, sim.Id()).NewEventSubscriber())
	sim.Subscribe(messaging.NewGameUpdateTx(uuid.Max, sim.Id()).NewEventSubscriber())

	go indexer.Start()
	sim.Start()
}
