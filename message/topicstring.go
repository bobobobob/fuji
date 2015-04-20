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

package message

import (
	"errors"
	"fmt"
	"regexp"
	"unicode/utf8"

	validator "gopkg.in/validator.v2"

	"github.com/shiguredo/fuji/inidef"
)

// init is automatically invoked at initial time.
func init() {
	validator.SetValidationFunc("validtopic", inidef.ValidMqttPublishTopic)
}

// TopicString is an type which is represents MQTT Topic string.
type TopicString struct {
	Str string `validate:"max=32767,validtopic"`
}

func (topic TopicString) Sring() string {
	return fmt.Sprintf("Topic: %s", topic.Str)
}

// Validate validates Topic is valid for MQTT or not.
func (topic TopicString) Validate() error {
	if err := validator.Validate(&topic); err != nil {
		return err
	}

	if !utf8.ValidString(topic.Str) {
		return errors.New("not a valid UTF8 string")
	}
	reu0 := regexp.MustCompile("\u0000")
	if reu0.FindString(topic.Str) != "" {
		return errors.New("topic should NOT include \\U0000 character")
	}
	rewild := regexp.MustCompile("[+#]+")
	if rewild.FindString(topic.Str) != "" {
		return errors.New("should not MQTT pub-topic include wildard character")
	}
	return nil
}
