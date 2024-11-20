package messaging

import (
	"context"
	"fmt"

	"github.com/CoralCoralCoralCoral/simulation-engine/protos/protos"
	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
)

const DAY_CHUNKS = 8

type GameUpdateTx struct {
	api_id  uuid.UUID
	sim_id  uuid.UUID
	updates chan *protos.GameUpdate
}

func NewGameUpdateTx(api_id, sim_id uuid.UUID) *GameUpdateTx {
	conn, err := amqp091.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "failed to connect to rabbit")

	ch, err := conn.Channel()
	failOnError(err, "failed to create channel")
	defer ch.Close()

	err = ch.ExchangeDeclare("game-updates", "topic", false, true, false, false, nil)
	failOnError(err, "failed to create exchange")

	tx := new(GameUpdateTx)

	tx.api_id = api_id
	tx.sim_id = sim_id
	tx.updates = make(chan *protos.GameUpdate, DAY_CHUNKS)

	// spin up some go routines to read from the updates chan and send updates to rabbit
	for range DAY_CHUNKS {
		go func() {
			ch, err := conn.Channel()
			failOnError(err, "failed to create channel for goroutine")
			defer ch.Close()

			for update := range tx.updates {
				tx.send(ch, update)
			}
		}()
	}

	return tx
}

func (tx *GameUpdateTx) NewEventSubscriber() func(event *protos.Event) {
	update := new(protos.GameUpdate)

	return func(event *protos.Event) {
		switch event.Type {
		case protos.EventType_EpochEnd:
			if payload, ok := event.Payload.(*protos.Event_EpochEnd); ok {
				if (payload.EpochEnd.Epoch*payload.EpochEnd.TimeStep)%(24/DAY_CHUNKS*60*60*1000) == 0 {
					tx.updates <- &protos.GameUpdate{
						AgentStateUpdates:    update.AgentStateUpdates,
						AgentLocationUpdates: update.AgentLocationUpdates,
						CommandsProcessed:    update.CommandsProcessed,
					}

					update = new(protos.GameUpdate)
				}
			}
		case protos.EventType_AgentStateUpdate:
			if payload, ok := event.Payload.(*protos.Event_AgentStateUpdate); ok {
				update.AgentStateUpdates = append(update.AgentStateUpdates, payload.AgentStateUpdate)
			}
		case protos.EventType_AgentLocationUpdate:
			if payload, ok := event.Payload.(*protos.Event_AgentLocationUpdate); ok {
				update.AgentLocationUpdates = append(update.AgentLocationUpdates, payload.AgentLocationUpdate)
			}
		case protos.EventType_CommandProcessed:
			if payload, ok := event.Payload.(*protos.Event_CommandProcessed); ok {
				update.CommandsProcessed = append(update.CommandsProcessed, payload.CommandProcessed)
			}
		}
	}
}

func (tx *GameUpdateTx) send(ch *amqp091.Channel, update *protos.GameUpdate) {
	routing_key := fmt.Sprintf("%s.%s", tx.api_id, tx.sim_id)

	body, err := proto.Marshal(update)
	failOnError(err, "failed to proto serialize event")

	err = ch.PublishWithContext(context.Background(),
		"game-updates", // exchange
		routing_key,    // routing key
		false,          // mandatory
		false,          // immediate
		amqp091.Publishing{
			ContentType: "application/x-protobuf",
			Body:        body,
		})

	failOnError(err, "Failed to publish message")
}
