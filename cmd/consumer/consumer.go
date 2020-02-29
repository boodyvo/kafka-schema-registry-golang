package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"

	"github.com/riferrei/srclient"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

func main() {
	var topic string
	var bootstrapService string
	var schemaRegistry string

	flag.StringVar(&topic, "topic", "test", "topic name for producing")
	flag.StringVar(&bootstrapService, "bs", "localhost", "list of kafka bootstrap service")
	flag.StringVar(&schemaRegistry, "sr", "http://localhost:8081", "list of kafka bootstrap service")
	// 1) Create the consumer as you would
	// normally do using Confluent's Go client
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapService,
		"group.id":          "myGroup",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		panic(err)
	}
	err = c.SubscribeTopics([]string{topic}, nil)
	defer func() {
		_ = c.Close()
	}()
	if err != nil {
		log.Fatal(err)
	}

	// 2) Create a instance of the client to retrieve the schemas for each message
	schemaRegistryClient := srclient.CreateSchemaRegistryClient(schemaRegistry)

	for {
		msg, err := c.ReadMessage(-1)
		if err == nil {
			// 3) Recover the schema id from the message and use the
			// client to retrieve the schema from Schema Registry.
			// Then use it to deserialize the record accordingly.
			schemaID := binary.BigEndian.Uint32(msg.Value[1:5])
			schema, err := schemaRegistryClient.GetSchema(int(schemaID))
			if err != nil {
				panic(fmt.Sprintf("Error getting the schema with id '%d' %s", schemaID, err))
			}
			native, _, _ := schema.Codec().NativeFromBinary(msg.Value[5:])
			value, _ := schema.Codec().TextualFromNative(nil, native)
			fmt.Printf("Got message: %s\n", string(value))
		} else {
			fmt.Printf("Error consuming the message: %v (%v)\n", err, msg)
		}
	}
}