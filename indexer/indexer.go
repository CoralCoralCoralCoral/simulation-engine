package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/CoralCoralCoralCoral/simulation-engine/protos/protos"
	"github.com/elastic/go-elasticsearch/esapi"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
)

func Start() {
	conn, err := amqp091.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Error connecting to rabbit: %s", err)
	}

	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{
			"https://localhost:9200",
		},
		Username:               "elastic",
		Password:               os.Getenv("ES_PASSWORD"),
		CertificateFingerprint: os.Getenv("ES_CERT_FINGERPRINT"),
	})

	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	index_name := "game_updates"

	err = createIndex(es, index_name)
	if err != nil {
		log.Fatalf("Error creating the index: %s", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Error creating rabbit channel: %s", err)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare("game-updates", "topic", false, true, false, false, nil)
	if err != nil {
		log.Fatalf("Error declaring exchange: %s", err)
	}

	queue, err := ch.QueueDeclare(
		"es_consumer", // Queue name
		false,         // Durable (survives broker restarts)
		true,          // Auto-delete
		false,         // Exclusive
		false,         // No-wait
		nil,           // Arguments
	)

	if err != nil {
		log.Fatalf("Failed to declare a queue: %s", err)
	}

	err = ch.QueueBind(
		queue.Name,     // Queue name
		"*.*",          // Routing key (matches all messages with two parts in the routing key)
		"game-updates", // Exchange name
		false,          // No-wait
		nil,            // Arguments
	)

	if err != nil {
		log.Fatalf("Failed to bind queue: %s", err)
	}

	updates, err := consumeUpdates(queue.Name, ch)
	if err != nil {
		log.Fatalf("Error consuming updates: %s", err)
	}

	for update := range updates {
		go sendBulkRequest(es, index_name, update)
	}
}

func createIndex(es *elasticsearch.Client, index_name string) error {
	// check if index already exists
	// res, err := es.Indices.Exists([]string{index_name})
	// if err != nil {
	// 	return fmt.Errorf("error checking if the index exists: %w", err)
	// }
	// defer res.Body.Close()

	// if res.StatusCode == 200 {
	// 	// index already exists
	// 	return nil
	// }

	res, err := es.Indices.Delete([]string{index_name})
	if err != nil {
		log.Fatalf("Error deleting the index: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 404 && res.StatusCode != 200 {
		log.Fatalf("Unexpected response when deleting index: %s", res.Status())
	}

	// Define the mapping
	mapping := `{
		"mappings": {
			"properties": {
				"sim_id": { "type": "keyword" },
				"agent_id": { "type": "keyword" },
				"epoch": { "type": "long" },
				"update_type": { "type": "keyword" },
				"location_id": { "type": "keyword" },
				"location_coordinates": { "type": "geo_point" },
				"state": { "type": "keyword" }
			}
		}
	}`

	// Create the index
	res, err = es.Indices.Create(index_name, es.Indices.Create.WithBody(strings.NewReader(mapping)))
	if err != nil {
		return fmt.Errorf("error creating index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response from Elasticsearch: %s", res.String())
	}

	return nil
}

func consumeUpdates(queue_name string, ch *amqp091.Channel) (<-chan []Update, error) {
	msgs, err := ch.Consume(
		queue_name, // queue
		"",         // consumer
		true,       // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return nil, err
	}

	updates := make(chan []Update)
	go func() {
		defer close(updates)
		for msg := range msgs {
			sim_id, err := extractSimId(msg.RoutingKey)
			if err != nil {
				log.Println("invalid routing key detected")
				continue
			}

			var game_update protos.GameUpdate

			err = proto.Unmarshal(msg.Body, &game_update)
			if err != nil {
				log.Printf("Failed to proto deserialize message: %v\n", err)
				continue
			}

			update := make([]Update, 0, len(game_update.AgentStateUpdates)+len(game_update.AgentLocationUpdates))

			for _, value := range game_update.AgentStateUpdates {
				update = append(update, Update{
					SimId:      sim_id,
					Epoch:      value.Epoch,
					AgentId:    value.Id,
					UpdateType: "state",
					State:      value.State,
					LocationId: value.LocationId,
					LocationCoordinates: &Coordinates{
						Latitude:  value.LocationLat,
						Longitude: value.LocationLon,
					},
				})
			}

			// for _, value := range game_update.AgentLocationUpdates {
			// 	update = append(update, Update{
			// 		Epoch:      value.Epoch,
			// 		AgentId:    value.Id,
			// 		LocationId: value.LocationId,
			// 		LocationCoordinates: &Coordinates{
			// 			Latitude:  value.LocationLat,
			// 			Longitude: value.LocationLon,
			// 		},
			// 	})
			// }

			updates <- update
		}
	}()

	return updates, nil
}

func extractSimId(routing_key string) (string, error) {
	parts := strings.Split(routing_key, ".")
	if len(parts) < 2 {
		return "", errors.New("invalid routing key")
	}

	return parts[1], nil
}

func sendBulkRequest(es *elasticsearch.Client, index_name string, updates []Update) error {
	var buffer bytes.Buffer

	for _, update := range updates {
		// Use UpdateType to distinguish between different document types
		doc_id := fmt.Sprintf("%s-%d-%s", update.AgentId, update.Epoch, update.UpdateType)

		// Metadata for the bulk request
		meta := fmt.Sprintf(`{ "index" : { "_index" : "%s", "_id" : "%s" } }%s`,
			index_name, doc_id, "\n")
		buffer.WriteString(meta)

		// Document to be indexed
		doc, err := json.Marshal(update)
		if err != nil {
			return fmt.Errorf("failed to marshal update: %w", err)
		}
		buffer.Write(doc)
		buffer.WriteString("\n")
	}

	// Send the bulk request
	req := esapi.BulkRequest{
		Body: bytes.NewReader(buffer.Bytes()),
	}

	res, err := req.Do(context.Background(), es)
	if err != nil {
		return fmt.Errorf("failed to send bulk request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error response from Elasticsearch: %s", res.String())
	}

	return nil
}
