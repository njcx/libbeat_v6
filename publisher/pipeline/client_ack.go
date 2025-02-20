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

package pipeline

import (
	"time"

	"github.com/njcx/libbeat_v6/beat"
	"github.com/njcx/libbeat_v6/common/atomic"
)

type clientACKer struct {
	acker
	active atomic.Bool
}

func (p *Pipeline) makeACKer(
	canDrop bool,
	cfg *beat.ClientConfig,
	waitClose time.Duration,
) acker {
	var (
		bld   = p.ackBuilder
		acker acker
	)

	sema := p.eventSema
	switch {
	case cfg.ACKCount != nil:
		acker = bld.createCountACKer(canDrop, sema, cfg.ACKCount)
	case cfg.ACKEvents != nil:
		acker = bld.createEventACKer(canDrop, sema, cfg.ACKEvents)
	case cfg.ACKLastEvent != nil:
		cb := lastEventACK(cfg.ACKLastEvent)
		acker = bld.createEventACKer(canDrop, sema, cb)
	default:
		if waitClose <= 0 {
			return bld.createPipelineACKer(canDrop, sema)
		}
		acker = bld.createCountACKer(canDrop, sema, func(_ int) {})
	}

	if waitClose <= 0 {
		return acker
	}
	return newWaitACK(acker, waitClose)
}

func lastEventACK(fn func(interface{})) func([]interface{}) {
	return func(events []interface{}) {
		fn(events[len(events)-1])
	}
}

func (a *clientACKer) lift(acker acker) {
	a.active = atomic.MakeBool(true)
	a.acker = acker
}

func (a *clientACKer) Active() bool {
	return a.active.Load()
}

func (a *clientACKer) close() {
	a.active.Store(false)
	a.acker.close()
}

func (a *clientACKer) addEvent(event beat.Event, published bool) bool {
	if a.active.Load() {
		return a.acker.addEvent(event, published)
	}
	return false
}

func (a *clientACKer) ackEvents(n int) {
	a.acker.ackEvents(n)
}

func buildClientCountACK(
	pipeline *Pipeline,
	canDrop bool,
	sema *sema,
	mk func(*clientACKer) func(int, int),
) acker {
	guard := &clientACKer{}
	cb := mk(guard)
	guard.lift(makeCountACK(pipeline, canDrop, sema, cb))
	return guard
}

func buildClientEventACK(
	pipeline *Pipeline,
	canDrop bool,
	sema *sema,
	mk func(*clientACKer) func([]interface{}, int),
) acker {
	guard := &clientACKer{}
	guard.lift(newEventACK(pipeline, canDrop, sema, mk(guard)))
	return guard
}
