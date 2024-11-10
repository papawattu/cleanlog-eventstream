package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"math/rand"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func generateRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func main() {

	port := flag.String("port", "8090", "port to listen on")
	groupName := flag.String("group", "foo", "kafka consumer group name")
	bootstrapServers := flag.String("bootstrap-servers", "localhost:29092,localhost:39092,localhost:49092", "kafka bootstrap servers")
	flag.Parse()

	log.Printf("Listening on port %s", fmt.Sprintf(":%s", *port))

	server := &http.Server{
		Addr: fmt.Sprintf(":%s", *port),
	}

	sigChan := make(chan os.Signal, 1)

	http.HandleFunc("GET /eventstream/{topicName}", func(w http.ResponseWriter, r *http.Request) {

		topicName := r.PathValue("topicName")
		lastEventID := r.Header.Get("Last-Event-ID")

		if topicName == "" {
			http.Error(w, "topicName is required", http.StatusBadRequest)
			return
		}

		clientName := r.Header.Get("clientName")
		if clientName == "" {
			clientName = generateRandomString(10)
		}

		group := r.Header.Get("group")
		if group == "" {
			group = generateRandomString(10)
		}
		offset := r.Header.Get("offset")

		if offset == "" {
			offset = "earliest"
		}

		consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
			"bootstrap.servers":             *bootstrapServers,
			"group.id":                      group,
			"auto.offset.reset":             offset,
			"enable.auto.commit":            "false",
			"client.id":                     clientName,
			"partition.assignment.strategy": "roundrobin",
		})

		if err != nil {
			log.Fatal(err)
		}
		// Create a new server-sent event
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		consumer.SubscribeTopics([]string{topicName}, nil)

		if lastEventID != "" {
			log.Printf("Last event id: %s\n", lastEventID)
			offsetInf, err := strconv.Atoi(lastEventID)
			if err != nil {
				offsetInf = 0
			}
			o := kafka.Offset(offsetInf)

			err = consumer.Assign([]kafka.TopicPartition{
				{Topic: &topicName, Partition: int32(0), Offset: o},
			})

			consumer.ReadMessage(0)
			if err != nil {
				log.Printf("Failed to assign partition: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
			consumer.Commit()
		}
		log.Printf("Subscribed to topic %s client %s offset %s\n", topicName, clientName, offset)

		defer consumer.Close()

		run := true
		for run {

			select {
			case <-r.Context().Done():
				log.Println("Client connection closed")
				run = false
			case <-sigChan:
				log.Println("Shutting down consumer")
				run = false

			default:
				ev := consumer.Poll(10)

				switch e := ev.(type) {

				case *kafka.Message:
					//log.Printf("Id: %s\n", e.TopicPartition.Offset)
					fmt.Fprintf(w, "event: message\n")
					fmt.Fprintf(w, "data: %s\n", string(e.Value))
					fmt.Fprintf(w, "id: %s\n", e.TopicPartition.Offset)
					consumer.Commit()
					w.(http.Flusher).Flush()
				case kafka.Error:
					//log.Fatalf("Error: %v\n", e)
					log.Printf("Error: %v\n", e)
				default:

				}
			}
		}

		log.Println("HTTP client connection closed - Closing consumer")

	})

	// Wait for a signal to shutdown

	go func() {

		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")

		if err := server.Close(); err != nil {
			log.Fatalf("HTTP close error: %v", err)
		}
	}()
	log.Printf("Group name %s", *groupName)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
