// Main reference: https://www.elmelectronics.com/wp-content/uploads/2016/07/ELM327DS.pdf
package main

import (
	"encoding/hex"
	"github.com/tarm/serial"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	simulatorPortEnv = "SIMULATOR_PORT"
	baudRate         = 38400
	frequencyReading = 10
	maxBufferSize    = 50
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
		"vehicleSpeed": getParameter("0D", func(out []byte) float64 {
			return float64(out[0]) // km/h
		}),
		"engineRPM": getParameter("0C", func(out []byte) float64 {
			return 256*float64(out[0]) + float64(out[1])/4 // RPM
		}),
		"absoluteBarometricPressure": getParameter("33", func(out []byte) float64 {
			return float64(out[0]) // kPa
		}),
		"throttlePosition": getParameter("11", func(out []byte) float64 {
			return 100 * float64(out[0]) / 255 // %
		}),
		"traveledWithMalfunction": getParameter("21", func(out []byte) float64 {
			return 256*float64(out[0]) + float64(out[1]) // Km
		}),
		"runtimeSinceEngineStart ": getParameter("1F", func(out []byte) float64 {
			return float64(out[0])*256 + float64(out[1]) // seconds
		}),
		"ambientAirTemperature": getParameter("46", func(out []byte) float64 {
			return float64(out[0]) - 40 // Cº
		}),
	}

	setUpPort()

	defer func() {
		log.Println("Shutting application down...")
		err := port.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	for name, parameter := range parameters {
		go func(n string, p Parameter) {
			for {
				out := p.collectData()
				log.Println(n, "=", out)
			}
		}(name, *parameter)
	}

	wg.Wait()
}

func setUpPort() {
	c := &serial.Config{
		Name:        os.Getenv(simulatorPortEnv),
		Baud:        baudRate,
		ReadTimeout: time.Second / 5, // Datasheet from Elm cites this value
	}

	log.Println("Chosen Port:", c.Name)
	log.Println("Chosen Baud Rate:", c.Baud)

	var err error
	port, err = serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}

	//port.Write([]byte("ATZ\r"))
	//buf := make([]byte, maxBufferSize)
	//port.Read(buf) // Discharded
	//port.Flush()
}

// Factory for Parameters
func getParameter(pid string, formula func([]byte) float64) *Parameter {
	return &Parameter{pid, formula}
}

func (p *Parameter) collectData() float64 {
	port.Flush()

	showCurrentDataService, expectedResponseLines, carriageReturn := "01", "1", "\r"
	command := []byte(showCurrentDataService + p.pid + expectedResponseLines + carriageReturn)

	readingWritingMux.Lock() // before writing

	n, err := port.Write(command)
	if err != nil {
		log.Println("Error writting to port:", err)
	} else if n == 0 {
		log.Println("No data written")
	}

	buf := make([]byte, maxBufferSize)
	n, err = port.Read(buf)

	readingWritingMux.Unlock() // after reading

	if err != nil {
		log.Println("Error reading from port:", err)
		return 0
	} else if n == 0 {
		log.Println("No data received")
		return 0
	}

	dataBytes := getDataBytes(buf[:n], "41", p.pid)
	return p.formula(dataBytes)
}

func getDataBytes(in []byte, expectedService string, pid string) []byte {
	octectsTmp := strings.Trim(string(in), ">")
	octects := strings.Split(octectsTmp, " ")
	for i := range octects {
		octects[i] = strings.TrimSpace(octects[i])
	}

	dataIdx := findDataBytes(octects, expectedService, pid)
	if dataIdx == -1 {
		log.Println("Couldn't find data")
		return []byte{}
	}
	dataString := strings.Join(octects[dataIdx:], "")
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
