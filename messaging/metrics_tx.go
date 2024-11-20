package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/CoralCoralCoralCoral/simulation-engine/model"
	"github.com/CoralCoralCoralCoral/simulation-engine/protos/protos"
	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"
)

type MetricsTx struct {
	api_id uuid.UUID
	sim_id uuid.UUID
	ch     *amqp091.Channel
}

type Metrics struct {
	NewInfections        int
	NewRecoveries        int
	InfectedPopulation   int
	InfectiousPopulation int
	ImmunePopulation     int
}

func NewMetricsTransmitter(api_id, sim_id uuid.UUID) *MetricsTx {
	conn, err := amqp091.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "failed to connect to rabbit")

	ch, err := conn.Channel()
	failOnError(err, "failed to create channel")

	err = ch.ExchangeDeclare("game-metrics", "topic", false, true, false, false, nil)
	failOnError(err, "failed to create exchange")

	return &MetricsTx{
		api_id,
		sim_id,
		ch,
	}
}

func (tx *MetricsTx) send(metrics *Metrics) {
	routing_key := fmt.Sprintf("%s.%s", tx.api_id, tx.sim_id)

	body, err := json.Marshal(metrics)
	failOnError(err, "failed to json serialize event")

	err = tx.ch.PublishWithContext(context.Background(),
		"game-metrics", // exchange
		routing_key,    // routing key
		false,          // mandatory
		false,          // immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		})

	failOnError(err, "Failed to publish message")
}

func (tx *MetricsTx) NewEventSubscriber() func(event *protos.Event) {
	metrics := new(Metrics)

	return func(event *protos.Event) {
		switch event.Type {
		case protos.EventType_EpochEnd:
			if payload, ok := event.Payload.(*protos.Event_EpochEnd); ok {
				if (payload.EpochEnd.Epoch*payload.EpochEnd.TimeStep)%(24*60*60*1000) != 0 {
					return
				}

				tx.send(metrics)
				metrics.print(payload.EpochEnd.GetTime().String())
				metrics.reset()
			}
		case protos.EventType_AgentStateUpdate:
			if payload, ok := event.Payload.(*protos.Event_AgentStateUpdate); ok {
				switch payload.AgentStateUpdate.State {
				case string(model.Infected):
					metrics.NewInfections += 1
					metrics.InfectedPopulation += 1
				case string(model.Infectious):
					metrics.InfectiousPopulation += 1
				case string(model.Immune):
					metrics.ImmunePopulation += 1
					metrics.NewRecoveries += 1
					metrics.InfectedPopulation -= 1
					metrics.InfectiousPopulation -= 1
				case string(model.Susceptible):
					metrics.ImmunePopulation -= 1
				default:
					panic("this should not be possible")
				}
			}
		default:
			// ignore other types of events
		}
	}
}

func (metrics *Metrics) reset() {
	metrics.NewInfections = 0
	metrics.NewRecoveries = 0
}

func (metrics *Metrics) print(date string) {
	fmt.Print("\033[H\033[2J")

	fmt.Printf("Epidemic state on %s\n", date)
	fmt.Printf("	New infections:			%d\n", metrics.NewInfections)
	fmt.Printf("	New recoveries:			%d\n", metrics.NewRecoveries)
	fmt.Printf("	Infected population:		%d\n", metrics.InfectedPopulation)
	fmt.Printf("	Infectious population:		%d\n", metrics.InfectiousPopulation)
	fmt.Printf("	Immune population:		%d\n", metrics.ImmunePopulation)
}
