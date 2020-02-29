package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/google/uuid"
	"github.com/riferrei/srclient"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type User struct {
	FavoriteNumber int    `json:"favorite_number"`
	Name           string `json:"name"`
}

func main() {
	var topic string
	var bootstrapService string
	var schemaRegistry string
	var schemaPath string
	flag.StringVar(&topic, "topic", "test", "topic name for producing")
	flag.StringVar(&bootstrapService, "bs", "localhost", "list of kafka bootstrap service")
	flag.StringVar(&schemaRegistry, "sr", "http://localhost:8081", "list of kafka bootstrap service")
	flag.StringVar(&schemaPath, "sp", "schema.avsc", "full path to schema")

	flag.Parse()

	// 1) Create the producer as you would normally do using Confluent's Go client
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": bootstrapService})
	if err != nil {
		panic(err)
	}
	defer p.Close()

	go func() {
		for event := range p.Events() {
			switch ev := event.(type) {
			case *kafka.Message:
				//message := ev
				_ = ev
				if ev.TopicPartition.Error != nil {
					//fmt.Printf("Error delivering the message '%s'\n", message.Key)
				} else {
					//fmt.Printf("Message '%s' delivered successfully!\n", message.Key)
				}
			}
		}
	}()

	// 2) Fetch the latest version of the schema, or create a new one if it is the first
	schemaRegistryClient := srclient.CreateSchemaRegistryClient(schemaRegistry)
	//schemaRegistryClient.SetCredentials("apiKey", "apiSecret")
	schema, err := schemaRegistryClient.GetLatestSchema(topic, false)
	if schema == nil {
		schemaBytes, _ := ioutil.ReadFile(schemaPath)
		fmt.Println(string(schemaBytes))
		schema, err = schemaRegistryClient.CreateSchema(topic, string(schemaBytes), false)
		if err != nil {
			panic(fmt.Sprintf("Error creating the schema %s", err))
		}
	}
	schemaIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(schemaIDBytes, uint32(schema.ID()))

	// 3) Serialize the record using the schema provided by the client,
	// making sure to include the schema id as part of the record.
	for {
		time.Sleep(2 * time.Second)
		newComplexType := User{
			Name:           randomdata.SillyName(),
			FavoriteNumber: rand.Intn(10000),
		}
		fmt.Printf("sending message: %#v\n", newComplexType)
		value, _ := json.Marshal(newComplexType)
		native, _, _ := schema.Codec().NativeFromTextual(value)
		valueBytes, _ := schema.Codec().BinaryFromNative(nil, native)

		var recordValue []byte
		recordValue = append(recordValue, byte(0))
		recordValue = append(recordValue, schemaIDBytes...)
		recordValue = append(recordValue, valueBytes...)

		key, _ := uuid.NewUUID()
		err = p.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic: &topic, Partition: kafka.PartitionAny},
			Key: []byte(key.String()), Value: recordValue}, nil)

		if err != nil {
			fmt.Println("error: ", err)
		}

		p.Flush(15 * 1000)
	}
}