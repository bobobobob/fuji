// Copyright 2015-2016 Shiguredo Inc. <fuji@shiguredo.jp>
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
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"

	"github.com/shiguredo/fuji"
	"github.com/shiguredo/fuji/broker"
	"github.com/shiguredo/fuji/config"
	"github.com/shiguredo/fuji/gateway"
	MYHTTP "github.com/shiguredo/fuji/http"
)

// Fuji for TestHttpConnectLocalPub
func fujiHttpConnectLocalPub(t *testing.T, httpConfigStr string) {
	assert := assert.New(t)

	conf, err := config.LoadConfigByte([]byte(httpConfigStr))
	assert.Nil(err)
	commandChannel := make(chan string)
	go fuji.StartByFileWithChannel(conf, commandChannel)
	t.Logf("fuji started")
	time.Sleep(10 * time.Second)
	t.Logf("fuji wait completed")
	commandChannel <- "close"
	// wait to stop fuji
	time.Sleep(1 * time.Second)
}

// Echoback body JSON HTTP server
func httpEchoServer(t *testing.T, cmdChan chan string, expectedJsonBody string) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, expectedJsonBody)
		t.Logf("request arrived: %v\n", r)
	}))
	defer ts.Close()

	// get usable port number
	t.Logf("Address: %s\n", ts.URL)

	cmdChan <- ts.URL

	// wait operation complete
	<-cmdChan
}

// TestHttpConnectLocalPubSub
// 1. connect gateway to local broker
// 2. subscribe to HTTP request
// 3. check subscribe
// 4. issue HTTP request
// 5. get HTTP response
// 6. publish response
func TestHttpConnectPostLocalPubSub(t *testing.T) {
	expected := []string{
		`{"id":"aasfa","url":"`,
		`","method":"POST","body":{"a":"b"}}`,
		`{"a":"b"}`,
		`{"id":"aasfa","status":200,"body":{"a":"b"}}`,
	}

	var httpConfigStr = `
[gateway]

    name = "httppostconnect"

[[broker."mosquitto/1"]]

    host = "localhost"
    port = 1883
    topic_prefix = "prefix"

    retry_interval = 10

[http]
    broker = "mosquitto"
    qos = 0
    enabled = true
`
	generalTestProcess(t, httpConfigStr, expected)
}

func TestHttpConnectGetLocalPubSub(t *testing.T) {
	expected := []string{
		`{"id":"aasfa","url":"`,
		`/?a=b","method":"GET","body":{}}`,
		`{"a":"b"}`,
		`{"id":"aasfa","status":200,"body":{"a":"b"}}`,
	}

	var httpConfigStr = `
[gateway]

    name = "httpgetconnect"

[[broker."mosquitto/1"]]

    host = "localhost"
    port = 1883
    topic_prefix = "prefix"

    retry_interval = 10

[http]
    broker = "mosquitto"
    qos = 0
    enabled = true
`
	generalTestProcess(t, httpConfigStr, expected)
}

func TestHttpConnectBadURLGetLocalPubSub(t *testing.T) {
	expected := []string{
		`{"id":"aasfa","url":"badprefix`,
		`/?a=b","method":"GET","body":{}}`,
		`{"a":"b"}`,
		`{"id":"aasfa","status":` + strconv.Itoa(MYHTTP.InvalidResponseCode) + `,"body":{}}`,
	}

	var httpConfigStr = `
[gateway]

    name = "httpgetbadurlconnect"

[[broker."mosquitto/1"]]

    host = "localhost"
    port = 1883

    retry_interval = 10

[http]
    broker = "mosquitto"
    qos = 0
    enabled = true
`
	generalTestProcess(t, httpConfigStr, expected)
}

func generalTestProcess(t *testing.T, httpConfigStr string, expectedStr []string) {
	assert := assert.New(t)

	requestJson_pre, requestJson_post, expectedJsonBody, expectedJson := expectedStr[0], expectedStr[1], expectedStr[2], expectedStr[3]

	// start fuji
	go fujiHttpConnectLocalPub(t, httpConfigStr)
	time.Sleep(1 * time.Second)

	// start http server
	httpCmdChan := make(chan string)
	go httpEchoServer(t, httpCmdChan, expectedJsonBody)
	// wait for bootup
	listener := <-httpCmdChan
	t.Logf("http started at: %s", listener)

	// pub/sub test to broker on localhost
	// publised JSON messages confirmed by subscriber

	// get config
	conf, err := config.LoadConfigByte([]byte(httpConfigStr))
	assert.Nil(err)

	// get Gateway
	gw, err := gateway.NewGateway(conf)
	assert.Nil(err)

	// get Broker
	brokerList, err := broker.NewBrokers(conf, gw.BrokerChan)
	assert.Nil(err)

	// Setup MQTT pub/sub client to confirm published content.
	//
	subscriberChannel := make(chan [2]string)

	opts := MQTT.NewClientOptions()
	url := fmt.Sprintf("tcp://%s:%d", brokerList[0].Host, brokerList[0].Port)
	opts.AddBroker(url)
	opts.SetClientID(fmt.Sprintf("prefix%s", gw.Name))
	opts.SetCleanSession(false)

	client := MQTT.NewClient(opts)
	assert.Nil(err)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		assert.Nil(token.Error())
		t.Log(token.Error())
	}

	qos := 0
	requestTopic := fmt.Sprintf("%s/%s/http/request", brokerList[0].TopicPrefix, gw.Name)
	expectedTopic := fmt.Sprintf("%s/%s/http/response", brokerList[0].TopicPrefix, gw.Name)
	t.Logf("expetcted topic: %s\nexpected message%s", expectedTopic, expectedJson)
	client.Subscribe(expectedTopic, byte(qos), func(client *MQTT.Client, msg MQTT.Message) {
		subscriberChannel <- [2]string{msg.Topic(), string(msg.Payload())}
	})
	// publish JSON
	token := client.Publish(requestTopic, 0, false, requestJson_pre+listener+requestJson_post)
	token.Wait()

	// wait for 1 publication of dummy worker
	t.Logf("wait for 1 publication of dummy worker")
	select {
	case message := <-subscriberChannel:
		assert.Equal(expectedTopic, message[0])
		assert.Equal(expectedJson, message[1])
	case <-time.After(time.Second * 11):
		assert.Equal("subscribe completed in 11 sec", "not completed")
	}

	client.Disconnect(20)
	httpCmdChan <- "done"
	time.Sleep(1 * time.Second)
}
