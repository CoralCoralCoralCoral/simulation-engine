package main

import (
	"flag"
	"log"
	"os"

	"github.com/CoralCoralCoralCoral/simulation-engine/messaging"
	"github.com/CoralCoralCoralCoral/simulation-engine/model"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/rabbitmq/amqp091-go"
)

func main() {
	loadDevEnvIfSet()

	rmq_conn, err := amqp091.Dial(os.Getenv("RMQ_URI"))
	if err != nil {
		log.Fatalf("couldn't create connection to rabbit: %s", err)
	}
	defer rmq_conn.Close()

	init_rx := messaging.NewInitRx(rmq_conn)
	init_rx.OnReceive(func(api_id uuid.UUID, config model.Config) {
		sim := model.NewSimulation(config, model.NewDefaultEntityGenerator())

		event_tx := messaging.NewEventTx(rmq_conn, api_id, sim.Id())
		defer event_tx.Close()

		sim.Subscribe(event_tx.NewEventSubscriber())

		metrics_tx := messaging.NewMetricsTx(rmq_conn, api_id, sim.Id())
		defer metrics_tx.Close()

		sim.Subscribe(metrics_tx.NewEventSubscriber())

		command_rx := messaging.NewCommandRx(rmq_conn, sim.Id())
		defer command_rx.Close()

		go command_rx.OnReceive(sim.SendCommand)

		sim.Start()
	})
}

func loadDevEnvIfSet() {
	dev := flag.Bool("dev", false, "Run in development mode")
	flag.Parse()

	if *dev {
		log.Println("running in dev environment")
		if err := godotenv.Load(); err != nil {
			log.Fatal("Error loading .env file")
		}
	}
}
