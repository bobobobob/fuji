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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shiguredo/fuji/broker"
	"github.com/shiguredo/fuji/config"
)

func TestNewHttpDevice(t *testing.T) {
	assert := assert.New(t)

	configStr := `
[http]
    broker = "sango"
    qos = 1
    enabled = true
`
	conf, err := config.LoadConfigByte([]byte(configStr))
	assert.Nil(err)
	b1 := &broker.Broker{Name: "sango"}
	brokers := []*broker.Broker{b1}
	fmt.Printf("conf:%v\n", conf)
	fmt.Printf("conf.Sections:%v\n", conf.Sections)
	b, err := NewHttpDevice(conf.Sections[0], brokers, NewDeviceChannel())
	assert.Nil(err)
	assert.NotNil(b.Broker)
	assert.Equal(byte(1), b.QoS)
}

func TestNewHttpDeviceInvalidQoS(t *testing.T) {
	assert := assert.New(t)

	configStr := `
[http]
    broker = "sango"
    qos = -1
    enabled = true
`
	conf, err := config.LoadConfigByte([]byte(configStr))
	assert.Nil(err)
	b1 := &broker.Broker{Name: "sango"}
	brokers := []*broker.Broker{b1}
	_, err = NewHttpDevice(conf.Sections[0], brokers, NewDeviceChannel())
	assert.NotNil(err)
}

func TestNewHttpDeviceInvalidBroker(t *testing.T) {
	assert := assert.New(t)

	configStr := `
[http]
    broker = "doesNotExist"
    qos = 1
    enabled = true 
`
	conf, err := config.LoadConfigByte([]byte(configStr))
	assert.Nil(err)
	b1 := &broker.Broker{Name: "sango"}
	brokers := []*broker.Broker{b1}
	_, err = NewHttpDevice(conf.Sections[0], brokers, NewDeviceChannel())
	assert.NotNil(err)
}
