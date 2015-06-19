// Copyright 2015 Shiguredo Inc. <fuji@shiguredo.jp>
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

package main

import (
	"fmt"
	"testing"
	"time"

	MQTT "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/stretchr/testify/assert"

	"github.com/shiguredo/fuji"
	"github.com/shiguredo/fuji/broker"
	"github.com/shiguredo/fuji/device"
	"github.com/shiguredo/fuji/gateway"
	"github.com/shiguredo/fuji/inidef"
)

// TestRetainJustPublish tests
// 1. connect gateway to local broker
// 2. send data with retaind flag dummy device normaly
func TestRetainJustPublish(t *testing.T) {
	assert := assert.New(t)
	iniStr := `
	[gateway]
	
	    name = retainham
	
	[broker "local/1"]
	
	    host = localhost
	    port = 1883
	
	[device "doraretain/dummy"]
	
	    broker = local
	    qos = 0
	
	    interval = 10
	    payload = Hello world retain true.
	
	    type = EnOcean
	    retain = true
`
	conf, err := inidef.LoadConfigByte([]byte(iniStr))
	assert.Nil(err)
	commandChannel := make(chan string)
	go fuji.StartByFileWithChannel(conf, commandChannel)

	time.Sleep(2 * time.Second)
}

// TestRetainSubscribePublishClose
// 1. connect gateway to local broker
// 2. send data with retaind flag from dummy device
// 3. disconnect
// 4. reconnect
// 5. subscirbe and receive data
func TestRetainSubscribePublishClose(t *testing.T) {
	assert := assert.New(t)
	iniStr := `
	[gateway]
	
	    name = testRetainafterclose
	
	[broker "local/1"]
	
	    host = localhost
	    port = 1883
	
	[device "dora/dummy"]
	
	    broker = local
	    qos = 0
	
	    interval = 10
	    payload = Hello retained world to subscriber after close.
	
	    type = EnOcean
	    retain = true
`
	commandChannel := make(chan string)
	conf, err := inidef.LoadConfigByte([]byte(iniStr))
	assert.Nil(err)
	go fuji.StartByFileWithChannel(conf, commandChannel)

	gw, err := gateway.NewGateway(conf)
	if err != nil {
		t.Error("Cannot make Gateway")
	}

	brokerList, err := broker.NewBrokers(conf, gw.BrokerChan)
	if err != nil {
		t.Error("Cannot make BrokerList")
	}

	devChan := device.NewDeviceChannel()
	gw.DeviceChannels = append(gw.DeviceChannels, devChan)
	dummyDevice, err := device.NewDummyDevice(conf.Sections[3], brokerList, devChan)
	if err != nil {
		t.Error("Cannot make DummyDeviceList")
	}

	go func() {
		time.Sleep(2 * time.Second)

		// kill publisher
		gw.Stop()

		time.Sleep(2 * time.Second)

		subscriberChannel, err := setupRetainSubscriber(gw, brokerList[0], &dummyDevice)
		if err != inidef.Error("") {
			t.Error(err)
		}
		// check Retained message
		retainedMessage := <-subscriberChannel
		retainedTopic := retainedMessage[0]
		retainedPayload := retainedMessage[1]

		expectedTopic := fmt.Sprintf("%s/%s/%s/%s", brokerList[0].TopicPrefix, gw.Name, dummyDevice.Name, dummyDevice.Type)
		expectedPayload := dummyDevice.Payload

		assert.Equal(expectedTopic, retainedTopic)
		assert.Equal(expectedPayload, retainedPayload)
	}()
	time.Sleep(5 * time.Second)
}

// setupRetainSubscriber returnes channel in order to read messages with retained flag
func setupRetainSubscriber(gw *gateway.Gateway, broker *broker.Broker, dummyDevice *device.DummyDevice) (chan [2]string, inidef.Error) {
	// Setup MQTT pub/sub client to confirm published content.
	//
	messageOutputChannel := make(chan [2]string)

	opts := MQTT.NewClientOptions()
	brokerUrl := fmt.Sprintf("tcp://%s:%d", broker.Host, broker.Port)
	opts.AddBroker(brokerUrl)
	opts.SetClientID(gw.Name + "testSubscriber") // to distinguish MQTT client from publisher
	opts.SetCleanSession(false)
	opts.SetDefaultPublishHandler(func(client *MQTT.Client, msg MQTT.Message) {
		messageOutputChannel <- [2]string{msg.Topic(), string(msg.Payload())}
	})

	client := MQTT.NewClient(opts)
	if client == nil {
		return nil, inidef.Error("NewClient failed")
	}

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, inidef.Error(fmt.Sprintf("NewClient Start failed %q", token.Error()))
	}
	qos := 0
	retainedTopic := fmt.Sprintf("%s/%s/%s/%s", broker.TopicPrefix, gw.Name, dummyDevice.Name, dummyDevice.Type)
	client.Subscribe(retainedTopic, byte(qos), func(client *MQTT.Client, msg MQTT.Message) {
	})

	return messageOutputChannel, inidef.Error("")
}
