// Main reference: https://www.elmelectronics.com/wp-content/uploads/2016/07/ELM327DS.pdf
package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	emitter "github.com/icaropires/go/v2"
	"github.com/tarm/serial"
)

const (
	simulatorPortEnv = "SIMULATOR_PORT"
	mqttHostEnv      = "MQTT_HOST"
	mqttPortEnv      = "MQTT_PORT"
	mqttKeyEnv       = "MQTT_KEY"
	carNameEnv       = "CAR_NAME"
	baudRate         = 38400
	maxBufferSize    = 50
	samplesAmount    = 1
	timeBeforeRead   = 100 // ms // Time after writing and before reading
	readTimeout      = 1   // s
)

// Parameter is a parameter read from OBDII
type Parameter struct {
	pid     string // Hex
	formula func(pid []byte) float64
}

var port *serial.Port
var readingWritingMux sync.Mutex

func main() {
	// Consult https://en.wikipedia.org/wiki/OBD-II_PIDs for basic PIDS
	parameters := map[string]*Parameter{
		"engineLoad": GetParameter("04", func(out []byte) float64 {
			return 100 * float64(out[0]) / 255 // %
		}),
		"engineCoolanteTemperature": GetParameter("05", func(out []byte) float64 {
			return float64(out[0]) - 40 // Cº
		}),
		"engineRPM": GetParameter("0C", func(out []byte) float64 {
			return (256*float64(out[0]) + float64(out[1])) / 4 // RPM
		}),
		"vehicleSpeed": GetParameter("0D", func(out []byte) float64 {
			return float64(out[0]) // km/h
		}),
		"traveledWithMalfunction": GetParameter("21", func(out []byte) float64 {
			return 256*float64(out[0]) + float64(out[1]) // Km
		}),
		//"runtimeSinceEngineStart ": GetParameter("1F", func(out []byte) float64 {
		//	return float64(out[0])*256 + float64(out[1]) // seconds
		//}),
		//"ambientAirTemperature": GetParameter("46", func(out []byte) float64 {
		//	return float64(out[0]) - 40 // Cº
		//}),
	}

	setUpPort()

	defer func() {
		log.Println("Shutting application down...")
		err := port.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	mqttClient := getConnectedMqttClient()
	mqttKey := os.Getenv(mqttKeyEnv)
	carName := os.Getenv(carNameEnv)
	mqttChannelPrefix := "cars/" + carName + "/"

	var wg sync.WaitGroup
	wg.Add(1)

	supportedPids := prepareElm327()
	for name, parameter := range parameters {
		go func(n string, p Parameter) {
			count := 0
			samples := make([]float64, samplesAmount)

			for {
				pidBytes, err := hex.DecodeString(p.pid)
				pidInt := int(pidBytes[0])
				if !isPidSupported(supportedPids, pidInt) {
					log.Printf("PID: 0x%02X is not supported by this vehicle!", pidInt)
					break
				}

				out, err := p.collectData()
				if err == nil {
					samples[count] = out
					count++
				}

				if count == samplesAmount {
					mean := float64(0)
					for _, sample := range samples {
						mean += sample
					}
					mean /= samplesAmount
					log.Println(n, "=", mean)

					count = 0
					samples = make([]float64, samplesAmount)

					channel := mqttChannelPrefix + n
					outStr := strconv.FormatFloat(mean, 'f', 2, 64)
					err := mqttClient.Publish(mqttKey, channel, outStr)
					if err != nil {
						log.Println("[NETWORK][ERROR] Couldn't publish data: ", err)
					}
				}
			}
		}(name, *parameter)
	}

	wg.Wait()
}

func setUpPort() {
	c := &serial.Config{
		Name:        os.Getenv(simulatorPortEnv),
		Baud:        baudRate,
		ReadTimeout: readTimeout,
	}

	log.Println("Chosen Port:", c.Name)
	log.Println("Chosen Baud Rate:", c.Baud)

	var err error
	port, err = serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}
}

func prepareElm327() []int {
	port.Flush()

	log.Println("Reseting ELM327...")
	submitCommand([]byte("ATZ\r"), 1000)

	pids := getSupportedPids()
	printSupportedPids(pids)

	return pids
}

func isPidSupported(pids []int, pid int) bool {
	idx := sort.SearchInts(pids, pid)

	if idx == len(pids) {
		return false
	}

	if pids[idx] == pid {
		return true
	}

	return false
}

func getSupportedPids() []int {
	log.Println("Setting searching for protocol...")
	submitCommand([]byte("ATSP0\r"), 100)

	port.Flush()
	log.Println("Searching for protocols and supported pids (1-20)...")
	buf, _ := submitCommand([]byte("0100\r"), 7000)
	data := getDataBytes(buf, "41", "00")
	pids := parseSupportedPids(data, 0x01)

	supportsNext := true
	for pidSupport := 0x20; supportsNext; pidSupport += 0x20 {
		supportsNext = false

		for i := len(pids) - 1; i >= 0; i-- {
			if pids[i] == pidSupport {
				supportsNext = true
				break
			} else if pids[i] < pidSupport {
				break
			}
		}

		if supportsNext {
			log.Printf("Searching for supported pids (%02X-%02X)...", pidSupport+1, pidSupport+0x20)

			pidSupportStr := fmt.Sprintf("%02X", pidSupport)
			command := "01" + pidSupportStr + "\r"

			buf, _ := submitCommand([]byte(command), 200)
			data := getDataBytes(buf, "41", pidSupportStr)
			pids = append(pids, parseSupportedPids(data, pidSupport+1)...)
		}
	}

	return pids
}

func printSupportedPids(pids []int) {
	pidsStr := make([]string, 0)
	for _, pid := range pids {
		pidsStr = append(pidsStr, fmt.Sprintf("0x%02X", pid))
	}

	fmt.Println("Supported PIDS by the vehicle: " + strings.Join(pidsStr, ", "))
}

func parseSupportedPids(data []byte, offset int) []int {
	pids := make([]int, 0)
	for i, b := range data {
		for j := 7; j >= 0; j-- {
			if b&(1<<uint(j)) != 0 {
				pids = append(pids, offset+(i*8)+int(7-j))
			}
		}
	}

	return pids
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

// GetParameter is a Factory for building parameters
func GetParameter(pid string, formula func([]byte) float64) *Parameter {
	return &Parameter{pid, formula}
}

func (p *Parameter) collectData() (float64, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("[ERROR] Error collecting data: ", r)
		}
	}()

	showCurrentDataService, expectedResponseLines, carriageReturn := "01", "1", "\r"
	command := []byte(showCurrentDataService + p.pid + expectedResponseLines + carriageReturn)

	buf, err := submitCommand(command, timeBeforeRead)
	if err != nil {
		return 0, err
	}

	dataBytes := getDataBytes(buf, "41", p.pid)
	return p.formula(dataBytes), nil
}

func submitCommand(command []byte, delayBeforeRead time.Duration /* ms */) ([]byte, error) {
	defer readingWritingMux.Unlock()
	readingWritingMux.Lock() // before writing

	port.Flush()

	n, err := port.Write(command)
	if err != nil {
		log.Println("[ERROR] Couldn't write to port:", err)
		return []byte{}, err
	} else if n == 0 {
		msg := "[ERROR] No data written"
		log.Println(msg)
		return []byte{}, errors.New(msg)
	}

	time.Sleep(time.Millisecond * delayBeforeRead)

	buf := make([]byte, maxBufferSize)
	n, err = port.Read(buf)

	if err != nil {
		log.Println("[ERROR] Couldn't read from port:", err)
		return []byte{}, err
	} else if n == 0 {
		msg := "[ERROR] No data received"
		log.Println(msg)
		return []byte{}, errors.New(msg)
	}

	return buf[:n], nil
}

func getDataBytes(in []byte, expectedService string, pid string) []byte {
	octectsStr := string(in)

	posData := strings.LastIndex(octectsStr, expectedService+" "+pid)
	if posData == -1 {
		log.Println("Couldn't find any data")
		return []byte{}
	}
	octectsStr = octectsStr[posData:]

	endOfData := strings.Index(octectsStr, "\r")
	if endOfData == -1 {
		endOfData = len(octectsStr)
	}
	octectsStr = octectsStr[:endOfData]

	octectsStr = strings.TrimSpace(octectsStr)
	octects := strings.Split(octectsStr, " ")
	for i := range octects {
		octects[i] = strings.TrimSpace(octects[i])
	}

	dataString := strings.Join(octects[2:], "")
	dataBytes, err := hex.DecodeString(dataString)
	if err != nil {
		log.Println("Invalid data: ", err)
		return []byte{}
	}

	return dataBytes
}

// Expected service equal to 40 + used service
func findDataBytes(octects []string, expectedService string, pid string) int {
	octectsLen := len(octects)

	for i := range octects {
		// >= 2 to have at least one byte of data
		// Example: [41, 0D, myData]
		if i >= 2 && octects[octectsLen-i-2] == expectedService && octects[octectsLen-i-1] == pid {
			return octectsLen - i
		}
	}

	return -1
}
