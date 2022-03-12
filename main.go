package rpiif

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/stianeikeland/go-rpio"
)

// Only used internally
type pulse struct {
	Value  uint8
	Length int64
}

// Main module struct
type IfScanner struct {
	Pin         rpio.Pin
	Initialized bool
	Scanning    bool
}

var (
	Scanner = IfScanner{
		Pin:         0,
		Initialized: false,
		Scanning:    false,
	}
)

// The module's errors
var (
	ErrInitialized      = errors.New("cannot initialize scanner: scanner is already initialized")
	ErrNotInitialized   = errors.New("cannot scan: not initialized: use Setup() before scanning")
	ErrAlreadyScanning  = errors.New("cannot concurrently scan: wait until scanning is finished before starting another scan")
	ErrCannotInitialize = errors.New("failed to initialize: hardware failure")
)

// Scans for received codes, this method is blocking
// Can return errors ErrNotInitialized or ErrAlreadyScanning
// Returns the received code as a hexadecimal string
func (scanner *IfScanner) Scan() (string, error) {
	if scanner.Scanning {
		return "", ErrAlreadyScanning
	}
	if !scanner.Initialized {
		return "", ErrNotInitialized
	}
	scanner.Scanning = true
	var count1 int = 0
	var command []pulse = make([]pulse, 0)
	var binary = make([]byte, 0)
	binary = append(binary, 1)
	var previousValue uint8 = 0
	var value = uint8(scanner.Pin.Read())
	for value == 1 {
		value = uint8(scanner.Pin.Read())
		time.Sleep(time.Microsecond * 100)
	}
	startTime := time.Now()
	for {
		time.Sleep(time.Nanosecond * 50)
		if value != previousValue {
			now := time.Now()
			pulseLength := now.Sub(startTime)
			startTime = now
			command = append(command, pulse{Value: previousValue, Length: pulseLength.Microseconds()})
		}
		if value == 1 {
			count1 += 1
		} else {
			count1 = 0
		}
		if count1 > 10000 {
			break
		}
		previousValue = value
		value = uint8(scanner.Pin.Read())
	}
	for _, item := range command {
		if item.Value == 1 {
			if item.Length > 1000 {
				binary = append(binary, 0)
				binary = append(binary, 1)
			} else {
				binary = append(binary, 0)
			}
		}
	}
	if len(binary) > 34 {
		binary = binary[:34]
	}
	res := ""
	for _, v := range binary {
		if v == 1 {
			res += "1"
		} else {
			res += "0"
		}
	}
	result := new(big.Int)
	result.SetString(res, 2)
	scanner.Scanning = false
	return fmt.Sprintf("%x", result), nil
}

// Initializes the scanner and binds it to a certain pin
// Example: `scanner.Setup(4)`
// This function has to be called before using `scanner.Scan()`
func (scanner *IfScanner) Setup(pinNumber uint8) error {
	if scanner.Initialized {
		return ErrInitialized
	}
	if err := rpio.Open(); err != nil {
		return err
	}
	// Set the pin to input
	scanner.Pin = rpio.Pin(pinNumber)
	scanner.Pin.Input()
	scanner.Initialized = true
	return nil
}
