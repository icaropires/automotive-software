/*Waits indefinitely for car messages.*/
package main

import (
	"log"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	emitter "github.com/icaropires/go/v2"
)

const (
	mqttHostEnv       = "MQTT_HOST"
	mqttPortEnv       = "MQTT_PORT"
	mqttKeyEnv        = "MQTT_KEY"
	mqttChannelPrefix = "cars/+/"
	outputFolderName  = "cars_output"
)

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	client := getConnectedMqttClient()
	key := os.Getenv(mqttKeyEnv)

	os.MkdirAll(outputFolderName, os.ModePerm)

	var wg sync.WaitGroup
	wg.Add(1)

	client.Subscribe(key, mqttChannelPrefix, func(_ *emitter.Client, msg emitter.Message) {
		SaveMsg(msg)
	})

	wg.Wait()
}

// SaveMsg gets messages sent by cars and save them, one folder per car, one file per parameter
func SaveMsg(msg emitter.Message) {
	topic := strings.Split(msg.Topic(), "/")

	const (
		prefixIdx = iota
		carNameIdx
		parameterNameIdx
	)

	carFolderPath := path.Join(outputFolderName, topic[carNameIdx])
	os.MkdirAll(carFolderPath, os.ModePerm)

	parameterFilePath := path.Join(carFolderPath, topic[parameterNameIdx])
	parameterFile, err := os.OpenFile(parameterFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		panic(err)
	}

	defer parameterFile.Close()

	log.SetOutput(parameterFile)
	log.Println(string(msg.Payload()))
	log.SetOutput(os.Stderr)
}

func getConnectedMqttClient() *emitter.Client {
	mqttHost := "tcp://" + os.Getenv(mqttHostEnv) + ":" + os.Getenv(mqttPortEnv)

	client, err := emitter.Connect(
		mqttHost,
		func(_ *emitter.Client, msg emitter.Message) {},
		emitter.WithConnectTimeout(time.Second*5),
		emitter.WithAutoReconnect(true),
		emitter.WithPingTimeout(time.Second*2),
	)

	if err != nil {
		log.Println("[NETWORK] [ERROR] Couldn't connect to MQTT broker: ", err)
	}

	client.OnConnect(func(_ *emitter.Client) {
		log.Println("[NETWORK] Connected with MQTT broker successfully")
	})
	client.OnDisconnect(func(_ *emitter.Client, err error) {
		log.Println("[NETWORK] Disconnected from MQTT broker: ", err)
	})
	client.OnError(func(_ *emitter.Client, err emitter.Error) {
		log.Println("[NETWORK] Error: ", err)
	})

	return client
}
