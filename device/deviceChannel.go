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
	"github.com/shiguredo/fuji/message"
)

const (
	MaxDeviceChanBufferSize = 20
)

type DeviceChannel struct {
	Chan chan message.Message
}

// NewDeviceChannel is a factory method to return DeviceChannel
func NewDeviceChannel() DeviceChannel {
	ch := DeviceChannel{
		Chan: make(chan message.Message, MaxDeviceChanBufferSize),
	}
	return ch
}

func NewDeviceChannels() []DeviceChannel {
	channels := []DeviceChannel{}
	return channels
}
