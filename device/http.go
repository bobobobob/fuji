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

package device

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	validator "gopkg.in/validator.v2"

	"github.com/shiguredo/fuji/broker"
	"github.com/shiguredo/fuji/config"
	"github.com/shiguredo/fuji/message"
)

type HttpDevice struct {
	Name           string `validate:"max=256,regexp=[^/]+,validtopic"`
	Type           string `validate:"max=256,regexp=[^/]+,validtopic"`
	Broker         []*broker.Broker
	BrokerName     string
	QoS            byte `validate:"min=0,max=2"`
	Retain         bool
	SubscribeTopic message.TopicString // fixed value
	PublishTopic   message.TopicString // fixed value
	DeviceChan     DeviceChannel       // GW -> device
}

func (device HttpDevice) String() string {
	var brokers []string
	for _, broker := range device.Broker {
		brokers = append(brokers, fmt.Sprintf("%s\n", broker))
	}
	return fmt.Sprintf("%#v", device)
}

// NewHttpDevice read config.ConfigSection and returnes HttpDevice.
// If config validation failed, return error
func NewHttpDevice(section config.ConfigSection, brokers []*broker.Broker, devChan DeviceChannel) (HttpDevice, error) {
	ret := HttpDevice{
		Name:       "http",
		Type:       "http",
		DeviceChan: devChan,
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
	} else {
		ret.QoS = byte(qos)
	}
	ret.Retain = false
	if values["retain"] == "true" {
		ret.Retain = true
	}

	// subscribe default topic
	ret.SubscribeTopic = message.TopicString{
		Str: strings.Join([]string{"http", "request"}, "/"),
	}
	// publish default topic
	ret.PublishTopic = message.TopicString{
		Str: strings.Join([]string{"http", "response"}, "/"),
	}

	if err := ret.Validate(); err != nil {
		return ret, err
	}

	return ret, nil
}

func (device *HttpDevice) Validate() error {
	validator := validator.NewValidator()
	validator.SetValidationFunc("validtopic", config.ValidMqttPublishTopic)
	if err := validator.Validate(device); err != nil {
		return err
	}
	return nil
}

type Request struct {
	Id     string `json:"id"`
	Url    string `json:"url"`
	Method string `json:"method"`
	Body   string `json:"body"`
}

type Response struct {
	Id     string  `json:"id"`
	Status float64 `json:"status"`
	Body   string  `json:"body"`
}

func httpCall(req Request, respPipe chan []byte) {
	var resp Response

	switch req.Method {
	case "POST":
		log.Infof("post body: %v\n", req.Body)
		reqbody := strings.NewReader(req.Body)
		httpresp, err := http.Post(req.Url, "vpplication/json;charset=utf-8", reqbody)
		defer httpresp.Body.Close()
		if err != nil {
			log.Error(err)
		}
		log.Infof("httpresp: %v\n", httpresp)
		log.Infof("Statuscode: %v\n", httpresp.StatusCode)
		respbodybuf, err := ioutil.ReadAll(httpresp.Body)

		var status float64
		status = 200
		// check error
		if err != nil {
			status = 502
			respbodybuf = []byte("")
		}
		// check response status
		if httpresp.StatusCode != 200 {
			status = 502
			respbodybuf = []byte("")
		}
		// make response data
		resp = Response{
			Id:     req.Id,
			Status: status,
			Body:   string(respbodybuf),
		}
		// TODO: implement GET
	default:
		// do nothing
		log.Error(errors.New("illegal method"))
		resp = Response{
			Id:     req.Id,
			Status: 502,
			Body:   "",
		}
	}
	// return response via chan
	log.Infof("resp: %v\n", resp)
	jsonbuf, err := json.Marshal(resp)
	if err != nil {
		log.Error(errors.New("Not a JSON response"))
		jsonbuf = []byte(`{"id": "` + req.Id + `", "status": 502, "body":"{}"}`)
	}
	log.Debugf("jsonbuf in string: %s\n", string(jsonbuf))
	respPipe <- jsonbuf
}

func (device HttpDevice) Start(channel chan message.Message) error {

	readPipe := make(chan []byte)

	log.Info("start http device")

	msgBuf := make([]byte, 65536)

	go func() error {
		for {
			select {
			case msgBuf = <-readPipe:
				log.Debugf("msgBuf to send: %v", msgBuf)
				msg := message.Message{
					Sender:     device.Name,
					Type:       device.Type,
					QoS:        device.QoS,
					Retained:   device.Retain,
					BrokerName: device.BrokerName,
					Body:       msgBuf,
				}
				channel <- msg
			case msg, _ := <-device.DeviceChan.Chan:
				log.Infof("msg topic:, %v / %v", msg.Topic, device.Name)
				if device.SubscribeTopic.Str == "" || !strings.HasSuffix(msg.Topic, device.SubscribeTopic.Str) {
					continue
				}
				log.Infof("msg reached to device, %v", msg)

				// compatible type to nested JSON
				var reqJson map[string]interface{}
				var jsonbuf []byte

				err := json.Unmarshal(msg.Body, &reqJson)
				// JSON error : 502
				if err != nil {
					log.Error(err)
					jsonbuf = []byte(`{"id": "", "status": 502, "body":"{}"}`)
					readPipe <- jsonbuf
					continue
				}
				bodyJson, err := json.Marshal(reqJson["body"].(map[string]interface{}))
				if err != nil {
					log.Error(err)
					jsonbuf = []byte(`{"id": "", "status": 502, "body":"{}"}`)
					readPipe <- jsonbuf
					continue
				}

				// issue HTTP request
				req := Request{
					Id:     reqJson["id"].(string),
					Url:    reqJson["url"].(string),
					Method: reqJson["method"].(string),
					Body:   string(bodyJson),
				}
				go httpCall(req, readPipe)
				log.Infof("http request issued")
			}
		}
	}()
	return nil
}

func (device HttpDevice) Stop() error {
	log.Infof("closing http: %v", device.Name)
	return nil
}

func (device HttpDevice) DeviceType() string {
	return "http"
}

func (device HttpDevice) AddSubscribe() error {
	if device.SubscribeTopic.Str == "" {
		return nil
	}
	for _, b := range device.Broker {
		b.AddSubscribed(device.SubscribeTopic, device.QoS)
	}
	return nil
}
