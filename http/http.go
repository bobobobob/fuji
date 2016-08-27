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

package http

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

type Http struct {
	Name           string `validate:"max=256,regexp=[^/]+,validtopic"`
	Type           string `validate:"max=256,regexp=[^/]+,validtopic"`
	Broker         []*broker.Broker
	BrokerName     string
	QoS            byte `validate:"min=0,max=2"`
	Retain         bool
	SubscribeTopic message.TopicString // fixed value
	PublishTopic   message.TopicString // fixed value
	HttpChan       HttpChannel         // GW -> http
}

type Request struct {
	Id     string `json:"id"`
	Url    string `json:"url"`
	Method string `json:"method"`
	Body   string `json:"body"`
}

type Response struct {
	Id     string          `json:"id"`
	Status float64         `json:"status"`
	Body   json.RawMessage `json:"body"`
}

const InvalidResponseCode = 502

func (device Http) String() string {
	var brokers []string
	for _, broker := range device.Broker {
		brokers = append(brokers, fmt.Sprintf("%s\n", broker))
	}
	return fmt.Sprintf("%#v", device)
}

// NewHttp read config.ConfigSection and returnes Http.
// If config validation failed, return error
func NewHttp(conf config.Config, brokers []*broker.Broker) (Http, []HttpChannel, error) {

	httpChan := NewHttpChannel()
	ret := Http{
		Name:     "http",
		Type:     "http",
		HttpChan: httpChan,
	}

	// search http section
	for _, section := range conf.Sections {
		if section.Type != "http" {
			continue
		}

		values := section.Values
		httpChannels := NewHttpChannels()
		httpChannels = append(httpChannels, httpChan)

		bname, ok := section.Values["broker"]
		if !ok {
			return ret, httpChannels, fmt.Errorf("broker does not set")
		}

		for _, b := range brokers {
			if b.Name == bname {
				ret.Broker = brokers
			}
		}
		if ret.Broker == nil {
			return ret, httpChannels, fmt.Errorf("broker does not exists: %s", bname)
		}
		ret.BrokerName = bname

		qos, err := strconv.Atoi(values["qos"])
		if err != nil {
			return ret, httpChannels, err
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
			return ret, httpChannels, err
		}

		return ret, httpChannels, nil
	}
	return ret, NewHttpChannels(), fmt.Errorf("no http section found in configuration")
}

func (device *Http) Validate() error {
	validator := validator.NewValidator()
	validator.SetValidationFunc("validtopic", config.ValidMqttPublishTopic)
	if err := validator.Validate(device); err != nil {
		return err
	}
	return nil
}

func httpCall(req Request, respPipe chan []byte) {
	var resp Response

	switch req.Method {
	case "POST":
		var status float64
		status = 200

		log.Debugf("post URL: %s\n", req.Url)
		log.Debugf("post body: %s\n", req.Body)
		reqbody := strings.NewReader(req.Body)
		httpresp, err := http.Post(req.Url, "application/json;charset=utf-8", reqbody)
		if err != nil {
			log.Errorf("POST response error: %s\n", err)
			resp = Response{
				Id:     req.Id,
				Status: InvalidResponseCode,
				Body:   json.RawMessage(`{}`),
			}
			break
		}

		log.Debugf("httpresp: %s\n", httpresp)
		log.Debugf("Statuscode: %s\n", httpresp.StatusCode)

		respbodybuf, err := ioutil.ReadAll(httpresp.Body)
		defer httpresp.Body.Close()

		// check error
		status = float64(httpresp.StatusCode)
		if err != nil {
			status = InvalidResponseCode
			respbodybuf = []byte("")
		}
		// make response data
		respbodystr := string(respbodybuf)
		log.Debugf("POST response body: %s", respbodystr)
		resp = Response{
			Id:     req.Id,
			Status: status,
			Body:   json.RawMessage(respbodystr),
		}
	case "GET":
		var statusget float64
		statusget = 200
		log.Debugf("request body: %s\n", req.Body)
		httpgetresp, err := http.Get(req.Url)
		if err != nil {
			log.Errorf("GET response error: %s\n", err)
			resp = Response{
				Id:     req.Id,
				Status: InvalidResponseCode,
				Body:   json.RawMessage(`{}`),
			}
			break
		}
		log.Debugf("httpgetresp: %s\n", httpgetresp)
		log.Debugf("Statuscode: %s\n", httpgetresp.StatusCode)
		statusget = float64(httpgetresp.StatusCode)
		respbodybuf, err := ioutil.ReadAll(httpgetresp.Body)
		// check error
		if err != nil {
			statusget = InvalidResponseCode
			respbodybuf = []byte("")
		}

		respbodystr := string(respbodybuf)
		log.Debugf("GET response body: %s", respbodystr)
		resp = Response{
			Id:     req.Id,
			Status: statusget,
			Body:   json.RawMessage(respbodystr),
		}
		defer httpgetresp.Body.Close()
	default:
		// do nothing
		log.Error(errors.New("illegal method"))
		resp = Response{
			Id:     req.Id,
			Status: InvalidResponseCode,
			Body:   json.RawMessage(`{}`),
		}
	}
	// return response via chan
	log.Debugf("resp: %s\n", resp)
	jsonbuf, err := json.Marshal(&resp)
	if err != nil {
		log.Error(errors.New("Not a JSON response"))
		jsonbuf = []byte(`{"id": "` + req.Id + `", "status": ` + strconv.Itoa(int(resp.Status)) + `, "body":"` + string(resp.Body) + `"}`)
	}
	log.Debugf("jsonbuf in string: %s\n", string(jsonbuf))
	respPipe <- jsonbuf
}

func (device Http) Start(channel chan message.Message) error {

	readPipe := make(chan []byte, 2)

	log.Info("start http device")

	msgBuf := make([]byte, 65536)

	go func() error {
		for {
			select {
			case msgBuf = <-readPipe:
				log.Debugf("msgBuf to send: %s", msgBuf)
				msg := message.Message{
					Sender:     device.Name,
					Type:       device.Type,
					QoS:        device.QoS,
					Retained:   device.Retain,
					BrokerName: device.BrokerName,
					Body:       msgBuf,
				}
				channel <- msg
			case msg, _ := <-device.HttpChan.Chan:
				log.Debugf("msg topic:, %s / %s", msg.Topic, device.Name)
				if device.SubscribeTopic.Str == "" || !strings.HasSuffix(msg.Topic, device.SubscribeTopic.Str) {
					continue
				}
				log.Debugf("msg reached to device, %s", msg)

				// nested JSON compatible type
				var reqJson map[string]interface{}
				var jsonbuf []byte

				err := json.Unmarshal(msg.Body, &reqJson)
				// JSON error would cause: 502 (InvalidResponseCode)
				if err != nil {
					log.Error(err)
					reqJsonError := Response{
						Id:     "",
						Status: InvalidResponseCode,
						Body:   json.RawMessage(`{}`),
					}
					reqJsonErrorStr, err := json.Marshal(&reqJsonError)
					if err != nil {
						log.Error(err)
						continue
					}
					jsonbuf = []byte(reqJsonErrorStr)
					readPipe <- jsonbuf
					continue
				}

				// check id first to return message
				originIdStr, idIsStr := reqJson["id"].(string)
				if !idIsStr {
					log.Error("id is nil. No way to return message")
					continue
				}

				// check required elements
				nilElements := false
				requiredElements := []string{"url", "method", "body"}
				for _, t := range requiredElements {
					log.Debugf("JSON element %v: %v", t, reqJson[t])
					if reqJson[t] == nil {
						nilErr := errors.New(fmt.Sprintf("element %v is not string or nil", t))
						log.Error(nilErr)
						nilElements = true
					}
				}
				if nilElements {
					log.Error("Some required JSON element are missing")
					reqJsonError := Response{
						Id:     originIdStr,
						Status: InvalidResponseCode,
						Body:   json.RawMessage(`{}`),
					}
					reqJsonErrorStr, err := json.Marshal(&reqJsonError)
					if err != nil {
						log.Error(err)
						continue
					}
					jsonbuf = []byte(reqJsonErrorStr)
					readPipe <- jsonbuf
					continue
				}

				// body shall be JSON object
				mapBodyJson, found := reqJson["body"].(map[string]interface{})
				if !found {
					reqJsonError := Response{
						Id:     originIdStr,
						Status: InvalidResponseCode,
						Body:   json.RawMessage(`{}`),
					}
					reqJsonErrorStr, err := json.Marshal(&reqJsonError)
					if err != nil {
						log.Error(err)
						continue
					}
					jsonbuf = []byte(reqJsonErrorStr)
					readPipe <- jsonbuf
					continue
				}

				bodyJson, err := json.Marshal(mapBodyJson)
				if err != nil {
					log.Error(err)
					respJsonError := Response{
						Id:     originIdStr,
						Status: InvalidResponseCode,
						Body:   json.RawMessage(`{}`),
					}
					respJsonErrorStr, err := json.Marshal(&respJsonError)
					if err != nil {
						log.Error(err)
						continue
					}
					jsonbuf = []byte(respJsonErrorStr)
					readPipe <- jsonbuf
					continue
				}

				// issue HTTP request
				req := Request{
					Id:     originIdStr,
					Url:    reqJson["url"].(string),
					Method: reqJson["method"].(string),
					Body:   string(bodyJson),
				}
				go httpCall(req, readPipe)
				log.Debugf("http request issued")
			}
		}
	}()
	return nil
}

func (device Http) Stop() error {
	log.Infof("closing http: %s", device.Name)
	return nil
}

func (device Http) DeviceType() string {
	return "http"
}

func (device Http) AddSubscribe() error {
	if device.SubscribeTopic.Str == "" {
		return nil
	}
	for _, b := range device.Broker {
		b.AddSubscribed(device.SubscribeTopic, device.QoS)
	}
	return nil
}
