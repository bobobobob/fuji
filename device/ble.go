// Copyright 2016 Shiguredo Inc. <fuji@shiguredo.jp>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package device

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/paypal/gatt"
	validator "gopkg.in/validator.v2"

	"github.com/shiguredo/fuji/broker"
	"github.com/shiguredo/fuji/config"
	"github.com/shiguredo/fuji/message"
)

const (
	StatusInit         = "init"
	StatusConnected    = "connected"
	StatusDisconnected = "disconnected"
)
const (
	maxMsgBufLen       = 256
	bleDefaultInterval = 10
)

type BLEDevice struct {
	Name           string `validate:"max=256,regexp=[^/]+,validtopic"`
	Broker         []*broker.Broker
	BrokerName     string
	QoS            byte   `validate:"min=0,max=2"`
	Type           string `validate:"max=256"`
	Interval       int    `validate:"min=0"`
	Retain         bool
	SubscribeTopic message.TopicString // initialized as ""
	DeviceChan     DeviceChannel       // GW -> device

	DeviceUUID          string `validate:"min=2"`
	ServiceUUID         string `validate:"min=2"`
	CharacteristicsUUID string `validate:"min=2"`

	status     string
	char       *gatt.Characteristic
	bledevice  *gatt.Device
	peripheral *gatt.Peripheral
	waitChan   chan error
	readChan   chan []byte
}

func (device BLEDevice) String() string {
	var brokers []string
	for _, broker := range device.Broker {
		brokers = append(brokers, fmt.Sprintf("%s\n", broker))
	}
	return fmt.Sprintf("%#v", device)
}

// NewBLEDevice read inidef.ConfigSection and returnes BLEDevice.
// If config validation failed, return error
func NewBLEDevice(section config.ConfigSection, brokers []*broker.Broker, devChan DeviceChannel) (BLEDevice, error) {
	ret := BLEDevice{
		Name:       section.Name,
		DeviceChan: devChan,
		Interval:   bleDefaultInterval,
		status:     StatusInit,
		readChan:   make(chan []byte),
	}
	values := section.Values
	bname, ok := section.Values["broker"]
	if !ok {
		return ret, fmt.Errorf("broker does not set")
	}

	for _, b := range brokers {
		if b.Name == bname {
			ret.Broker = brokers
		}
	}
	if ret.Broker == nil {
		return ret, fmt.Errorf("broker does not exists: %s", bname)
	}
	ret.BrokerName = bname

	qos, err := strconv.Atoi(values["qos"])
	if err != nil {
		return ret, err
	}

	ret.QoS = byte(qos)

	interval, err := strconv.Atoi(values["interval"])
	if err == nil {
		ret.Interval = interval
	}
	if ret.DeviceUUID, ok = values["device"]; !ok {
		return ret, fmt.Errorf("device should be set")
	}
	ret.DeviceUUID = strings.Replace(ret.DeviceUUID, "-", "", -1)

	if ret.ServiceUUID, ok = values["service"]; !ok {
		return ret, fmt.Errorf("service should be set")
	}
	if ret.CharacteristicsUUID, ok = values["characteristic"]; !ok {
		return ret, fmt.Errorf("characteristic should be set")
	}

	ret.Type = values["type"]
	ret.Retain = false
	if values["retain"] == "true" {
		ret.Retain = true
	}

	sub, ok := values["subscribe"]
	if ok && sub == "true" {
		ret.SubscribeTopic = message.TopicString{
			Str: strings.Join([]string{ret.Name, ret.Type, "subscribe"}, "/"),
		}
	}

	if err := ret.Validate(); err != nil {
		return ret, err
	}

	err = ret.init()
	if err != nil {
		return ret, err
	}

	return ret, nil
}

func (device *BLEDevice) Validate() error {
	validator := validator.NewValidator()
	if err := validator.SetValidationFunc("validtopic", config.ValidMqttPublishTopic); err != nil {
		return err
	}
	if err := validator.Validate(device); err != nil {
		return err
	}
	return nil
}

func (device *BLEDevice) onStateChanged(d gatt.Device, s gatt.State) {
	switch s {
	case gatt.StatePoweredOn:
		log.Debugf("BLE: scan start")
		d.Scan([]gatt.UUID{}, false)
		device.bledevice = &d
		return
	default:
		log.Debugf("BLE: state changed: %s", s)
		d.StopScanning()
	}
}
func (device *BLEDevice) onPeriphDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
	id := strings.ToUpper(device.DeviceUUID)
	log.Debugf("BLE: device found: %s", strings.ToUpper(p.ID()))

	if strings.ToUpper(p.ID()) != id {
		return
	}
	log.Debugf("BLE: device discovered. scan stopped and try to connect")

	p.Device().StopScanning()

	p.Device().Connect(p)
}

func (device *BLEDevice) onPeriphDisconnected(p gatt.Peripheral, err error) {
	log.Warnf("BLE: device disconnected. start to rescan")
	device.status = StatusDisconnected

	//	time.Sleep(100 * time.Millisecond)

	p.Device().Scan([]gatt.UUID{}, false)
	d := *device.bledevice
	if device.peripheral == nil {
		log.Warnf("BLE: device.peripheral is null")
	} else {
		d.CancelConnection(*device.peripheral)
	}
}

func (device *BLEDevice) onPeriphConnected(p gatt.Peripheral, err error) {
	log.Debugf("BLE: device connected.")

	sUUID, err := gatt.ParseUUID(device.ServiceUUID)
	if err != nil {
		device.waitChan <- fmt.Errorf("BLE: could not parse service UUID: %s", err)
		return
	}
	cUUID, err := gatt.ParseUUID(device.CharacteristicsUUID)
	if err != nil {
		device.waitChan <- fmt.Errorf("BLE: could not parse characteristics UUID: %s", err)
		return
	}

	services, err := p.DiscoverServices(nil)
	if err != nil {
		device.waitChan <- fmt.Errorf("discover services failed: %s", err)
		return
	}
	for _, service := range services {
		log.Debugf("service found: %s", service.UUID().String())
		if service.UUID().String() != sUUID.String() {
			continue
		}
		log.Infof("BLE: service %s found", device.ServiceUUID)

		chars, err := p.DiscoverCharacteristics(nil, service)
		if err != nil {
			err := fmt.Errorf("discover characteristics failed:%s", err)
			device.waitChan <- err
			return
		}
		for _, char := range chars {
			log.Debugf("characterist found: %s", char.UUID().String())
			if char.UUID().String() != cUUID.String() {
				continue
			}
			log.Infof("BLE: characteristics %s found", device.CharacteristicsUUID)
			device.peripheral = &p
			device.char = char
			if device.status == StatusInit {
				device.waitChan <- nil
			} else if device.status == StatusDisconnected {
				device.status = StatusConnected
			}
			go device.readLoop()
			return
		}
		device.waitChan <- errors.New("could not find specfied characteristics")
		return
	}
	device.waitChan <- errors.New("could not find specified service")
	return
}

func (device *BLEDevice) init() error {
	// create New BLE device with platform dependent gattOption.
	d, err := gatt.NewDevice(gattOption...)
	if err != nil {
		return fmt.Errorf("BLE: open BLE device failed: %s", err)
	}
	// Register handlers.
	d.Handle(
		gatt.PeripheralDiscovered(device.onPeriphDiscovered),
		gatt.PeripheralConnected(device.onPeriphConnected),
		gatt.PeripheralDisconnected(device.onPeriphDisconnected),
	)

	device.waitChan = make(chan error)

	err = d.Init(device.onStateChanged)
	if err != nil {
		return err
	}

	timeout := 20 // sec
	select {
	case err = <-device.waitChan:
		if err != nil {
			return fmt.Errorf("connect BLE failed: %s", err)
		}
	case <-time.After(time.Duration(timeout) * time.Second):
		return fmt.Errorf("connect BLE timed out")
	}

	log.Debugf("BLE: settings done")
	return nil
}

func (device *BLEDevice) readLoop() error {
	tick := time.Tick(time.Duration(device.Interval) * time.Second)

	for {
		select {
		case <-tick:
			if (device.char.Properties() & gatt.CharRead) != 0 {
				p := *device.peripheral
				timeout := make(chan []byte, 1)
				var b []byte
				go func() {
					b, err := p.ReadCharacteristic(device.char)
					if err != nil {
						log.Warnf("Failed to read characteristic: %s", err)
						timeout <- []byte{}
					}
					timeout <- b
				}()
				select {
				case b = <-timeout:
					if len(b) == 0 {
						continue
					}
				case <-time.After(5 * time.Second):
					log.Warnf("read characteristic timeout. exit read loop")
					return nil
				}
				device.readChan <- b

			}
		}
	}

	return nil
}

func (device *BLEDevice) write(buf []byte) error {
	p := *device.peripheral
	return p.WriteCharacteristic(device.char, buf, false) // noRsp is false?
}

func (device *BLEDevice) startLoop(msgChan chan message.Message) error {
	msgBuf := make([]byte, maxMsgBufLen)
	var ok bool
	for {
		select {
		case msgBuf, ok = <-device.readChan:
			if !ok {
				return fmt.Errorf("BLE: read pipe closed")
			}
			log.Debugf("BLE: msgBuf to send: %v", msgBuf)
			msg := message.Message{
				Sender:     device.Name,
				Type:       device.Type,
				QoS:        device.QoS,
				Retained:   true, // always true
				BrokerName: device.BrokerName,
				Body:       msgBuf,
			}
			msgChan <- msg
		case msg, _ := <-device.DeviceChan.Chan: // From subscribed
			if !strings.HasSuffix(msg.Topic, device.Name) {
				log.Debugf("BLE: subscibe msg Topic(%s) does not have deive.Name(%s)", msg.Topic, device.Name)
				continue
			}
			log.Infof("BLE: msg from sub/ topic:%v / %v", msg.Topic, device.Name)
			err := device.write(msg.Body)
			if err != nil {
				log.Errorf("BLE: could not write, %s", err)
			}
		}
	}
}

func (device BLEDevice) Start(msgChan chan message.Message) error {
	log.Debug("BLE: start device")

	go device.startLoop(msgChan)
	return nil
}

func (device *BLEDevice) disconnect() error {
	d := *device.bledevice
	if device.peripheral == nil {
		log.Warnf("BLE: device.peripheral is null in disconnect")
	} else {
		d.CancelConnection(*device.peripheral)
	}
	return nil
}
func (device BLEDevice) Stop() error {
	log.Warnf("closing BLE device: %v", device.Name)
	return device.disconnect()
}

func (device BLEDevice) DeviceType() string {
	return "ble"
}

func (device BLEDevice) AddSubscribe() error {
	if device.SubscribeTopic.Str == "" {
		return nil
	}
	for _, b := range device.Broker {
		err := b.AddSubscribed(device.SubscribeTopic, device.QoS)
		if err != nil {
			return err
		}

	}
	return nil
}
