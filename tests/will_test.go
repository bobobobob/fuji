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
	"github.com/shiguredo/fuji/gateway"
	"github.com/shiguredo/fuji/inidef"
)

// TestWillJustPublish tests
// 1. connect localhost broker with will message
// 2. send data from a dummy device
// 3. disconnect
func TestWillJustPublish(t *testing.T) {
	assert := assert.New(t)

	iniStr := `
	[gateway]
	    name = willjustpublishham
	[broker "local/1"]
	    host = localhost
	    port = 1883
	    will_message = no letter is good letter.
	[device "dora/dummy"]
	    broker = local
	    qos = 0
	    interval = 10
	    payload = Hello will just publish world.
	    type = EnOcean
`
	conf, err := inidef.LoadConfigByte([]byte(iniStr))
	assert.Nil(err)
	commandChannel := make(chan string)
	go fuji.StartByFileWithChannel(conf, commandChannel)
	time.Sleep(5 * time.Second)

	//	fuji.Stop()
}

// TestWillSubscribePublishClose
// 1. connect subscriber and publisher to localhost broker with will message
// 2. send data from a dummy device
// 3. force disconnect
// 4. check subscriber receives will message
func TestWillSubscribePublishClose(t *testing.T) {
	assert := assert.New(t)

	iniStr := `
	[gateway]
	    name = testwillafterclose
	[broker "local/1"]
	    host = localhost
	    port = 1883
	    will_message = good letter is no letter.
	[device "dora/dummy"]
	    broker = local
	    qos = 0
	    interval = 10
	    payload = Hello will just publish world.
	    type = EnOcean
`
	ok := genericWillTestDriver(t, iniStr, "/testwillafterclose/will", []byte("good letter is no letter."))
	assert.True(ok, "Failed to receive Will message")
}

// TestWillSubscribePublishCloseEmpty
// 1. connect subscriber and publisher to localhost broker with will message
// 2. send data from a dummy device
// 3. force disconnect
// 4. check subscriber receives will message
func TestWillSubscribePublishCloseEmpty(t *testing.T) {
	iniStr := `
	[gateway]
	    name = testwillaftercloseemptywill
	[broker "local/1"]
	    host = localhost
	    port = 1883
	    will_message = 
	[device "dora/dummy"]
	    broker = local
	    qos = 0
	    interval = 10
	    payload = Hello will just publish world.
	    type = EnOcean
`
	ok := genericWillTestDriver(t, iniStr, "/testwillaftercloseemptywill/will", []byte{})
	if !ok {
		t.Error("Failed to receive Empty Will message")
	}
}

func TestWillSubscribePublishBinaryWill(t *testing.T) {
	iniStr := `
	[gateway]
	    name = binary
	[broker "local/1"]
	    host = localhost
	    port = 1883
	    will_message = \x01\x02
	[device "dora/dummy"]
	    broker = local
	    qos = 0
	    interval = 10
	    payload = Hello will just publish world.
	    type = EnOcean
`
	ok := genericWillTestDriver(t, iniStr, "/binary/will", []byte{1, 2})
	if !ok {
		t.Error("Failed to receive Empty Will message")
	}
}

// genericWillTestDriver
// 1. read config string
// 2. connect subscriber and publisher to localhost broker with will message
// 3. send data from a dummy device
// 4. force disconnect
// 5. check subscriber receives will message

func genericWillTestDriver(t *testing.T, iniStr string, expectedTopic string, expectedPayload []byte) (ok bool) {
	assert := assert.New(t)

	conf, err := inidef.LoadConfigByte([]byte(iniStr))
	assert.Nil(err)
	commandChannel := make(chan string)
	go fuji.StartByFileWithChannel(conf, commandChannel)

	gw, err := gateway.NewGateway(conf)
	assert.Nil(err)

	brokers, err := broker.NewBrokers(conf, gw.BrokerChan)
	assert.Nil(err)

	go func() {
		time.Sleep(1 * time.Second)

		subscriberChannel, err := setupWillSubscriber(gw, brokers[0])
		if err != inidef.Error("") {
			t.Error(err)
		}

		time.Sleep(1 * time.Second)

		// kill publisher
		brokers[0].FourceClose()
		fmt.Println("broker killed for getting will message")

		// check will message
		willMsg := <-subscriberChannel

		assert.Equal(expectedTopic, willMsg.Topic())
		assert.Equal(expectedPayload, willMsg.Payload())
		assert.Equal(byte(0), willMsg.Qos())
	}()
	time.Sleep(3 * time.Second)
	ok = true
	return ok
}

// setupWillSubscriber start subscriber process and returnes a channel witch can receive will message.
func setupWillSubscriber(gw *gateway.Gateway, broker *broker.Broker) (chan MQTT.Message, inidef.Error) {
	// Setup MQTT pub/sub client to confirm published content.
	//
	messageOutputChannel := make(chan MQTT.Message)

	opts := MQTT.NewClientOptions()
	brokerUrl := fmt.Sprintf("tcp://%s:%d", broker.Host, broker.Port)
	opts.AddBroker(brokerUrl)
	opts.SetClientID(gw.Name + "testSubscriber") // to distinguish MQTT client from publisher
	opts.SetCleanSession(false)
	opts.SetDefaultPublishHandler(func(client *MQTT.Client, msg MQTT.Message) {
		messageOutputChannel <- msg
	})

	client := MQTT.NewClient(opts)
	if client == nil {
		return nil, inidef.Error("NewClient failed")
	}
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, inidef.Error(fmt.Sprintf("NewClient Start failed %q", token.Error()))
	}

	qos := 0
	// assume topicPrefix == ""
	willTopic := fmt.Sprintf("/%s/will", gw.Name)
	client.Subscribe(willTopic, byte(qos), func(client *MQTT.Client, msg MQTT.Message) {
		messageOutputChannel <- msg
	})

	return messageOutputChannel, inidef.Error("")
}
