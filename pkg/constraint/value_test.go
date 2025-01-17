/*
Copyright 2021 The tKeel Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package constraint

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseConfigsFrom(t *testing.T) {
	now := time.Now().UnixNano() / 1e6
	data := map[string]interface{}{
		"id":             "property1",
		"type":           "int",
		"weight":         20,
		"enabled":        true,
		"enabled_search": true,
		"description":    "property instance.",
		"last_time":      now,
		"define":         map[string]interface{}{"max": 200},
	}

	cfg := Config{
		ID:                "property1",
		Type:              "int",
		Weight:            20,
		Enabled:           true,
		EnabledSearch:     true,
		EnabledTimeSeries: false,
		Description:       "property instance.",
		LastTime:          now,
		Define: map[string]interface{}{
			"max": 200,
		},
	}

	result, err := ParseConfigsFrom(data)
	assert.Nilf(t, err, "parse successful.")
	assert.Equal(t, cfg, result)
}
