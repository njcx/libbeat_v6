// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build !integration

package console

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/njcx/libbeat_v6/beat"
	"github.com/njcx/libbeat_v6/common"
	"github.com/njcx/libbeat_v6/common/fmtstr"
	"github.com/njcx/libbeat_v6/outputs"
	"github.com/njcx/libbeat_v6/outputs/codec"
	"github.com/njcx/libbeat_v6/outputs/codec/format"
	"github.com/njcx/libbeat_v6/outputs/codec/json"
	"github.com/njcx/libbeat_v6/outputs/outest"
	"github.com/njcx/libbeat_v6/publisher"
)

// capture stdout and return captured string
func withStdout(fn func()) (string, error) {
	stdout := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	os.Stdout = w
	defer func() {
		os.Stdout = stdout
	}()

	outC := make(chan string)
	go func() {
		// capture all output
		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		r.Close()
		outC <- buf.String()
	}()

	fn()
	w.Close()
	result := <-outC
	return result, err
}

// TODO: add tests with other formatstr codecs

func TestConsoleOutput(t *testing.T) {
	tests := []struct {
		title    string
		codec    codec.Codec
		events   []beat.Event
		expected string
	}{
		{
			"single json event (pretty=false)",
			json.New(false, true, "1.2.3"),
			[]beat.Event{
				{Fields: event("field", "value")},
			},
			"{\"@timestamp\":\"0001-01-01T00:00:00.000Z\",\"@metadata\":{\"beat\":\"test\",\"type\":\"doc\",\"version\":\"1.2.3\"},\"field\":\"value\"}\n",
		},
		{
			"single json event (pretty=true)",
			json.New(true, true, "1.2.3"),
			[]beat.Event{
				{Fields: event("field", "value")},
			},
			"{\n  \"@timestamp\": \"0001-01-01T00:00:00.000Z\",\n  \"@metadata\": {\n    \"beat\": \"test\",\n    \"type\": \"doc\",\n    \"version\": \"1.2.3\"\n  },\n  \"field\": \"value\"\n}\n",
		},
		// TODO: enable test after update fmtstr support to beat.Event
		{
			"event with custom format string",
			format.New(fmtstr.MustCompileEvent("%{[event]}")),
			[]beat.Event{
				{Fields: event("event", "myevent")},
			},
			"myevent\n",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.title, func(t *testing.T) {
			batch := outest.NewBatch(test.events...)
			lines, err := run(test.codec, batch)
			assert.Nil(t, err)
			assert.Equal(t, test.expected, lines)

			// check batch correctly signalled
			if !assert.Len(t, batch.Signals, 1) {
				return
			}
			assert.Equal(t, outest.BatchACK, batch.Signals[0].Tag)
		})
	}
}

func run(codec codec.Codec, batches ...publisher.Batch) (string, error) {
	return withStdout(func() {
		c, _ := newConsole("test", outputs.NewNilObserver(), codec)
		for _, b := range batches {
			c.Publish(b)
		}
	})
}

func event(k, v string) common.MapStr {
	return common.MapStr{k: v}
}
