package main

import (
	"encoding/json"

	"github.com/CoralCoralCoralCoral/simulation-engine/model"
	"github.com/google/uuid"
)

func main() {
	pathogen := model.Pathogen{
		IncubationPeriod:   [2]float64{3 * 24 * 60 * 60 * 1000, 8 * 60 * 60 * 1000},
		RecoveryPeriod:     [2]float64{7 * 24 * 60 * 60 * 1000, 8 * 60 * 60 * 1000},
		ImmunityPeriod:     [2]float64{330 * 24 * 60 * 60 * 1000, 90 * 24 * 60 * 60 * 1000},
		QuantaEmissionRate: [2]float64{250, 100},
	}

	config := model.Config{
		Id:        uuid.New(),
		TimeStep:  15 * 60 * 1000,
		NumAgents: 150000,
		Pathogen:  pathogen,
	}

	body, err := json.Marshal(config)
	if err != nil {
		panic("AAAAH")
	}

	println(string(body))
}
