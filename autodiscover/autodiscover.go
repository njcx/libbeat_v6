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

package autodiscover

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/njcx/libbeat_v6/autodiscover/meta"
	"github.com/njcx/libbeat_v6/beat"
	"github.com/njcx/libbeat_v6/cfgfile"
	"github.com/njcx/libbeat_v6/common"
	"github.com/njcx/libbeat_v6/common/bus"
	"github.com/njcx/libbeat_v6/common/reload"
	"github.com/njcx/libbeat_v6/logp"
)

const (
	debugK = "autodiscover"

	// If a config reload fails after a new event, a new reload will be run after this period
	retryPeriod = 10 * time.Second
)

// TODO autodiscover providers config reload

// Adapter must be implemented by the beat in order to provide Autodiscover
type Adapter interface {

	// CreateConfig generates a valid list of configs from the given event, the received event will have all keys defined by `StartFilter`
	CreateConfig(bus.Event) ([]*common.Config, error)

	// RunnerFactory provides runner creation by feeding valid configs
	cfgfile.RunnerFactory

	// EventFilter returns the bus filter to retrieve runner start/stop triggering events
	EventFilter() []string
}

// Autodiscover process, it takes a beat adapter and user config and runs autodiscover process, spawning
// new modules when any configured providers does a match
type Autodiscover struct {
	bus             bus.Bus
	defaultPipeline beat.Pipeline
	adapter         Adapter
	providers       []Provider
	configs         map[string]map[uint64]*reload.ConfigWithMeta
	runners         *cfgfile.RunnerList
	meta            *meta.Map

	listener bus.Listener
}

// NewAutodiscover instantiates and returns a new Autodiscover manager
func NewAutodiscover(name string, pipeline beat.Pipeline, adapter Adapter, config *Config) (*Autodiscover, error) {
	// Init Event bus
	bus := bus.New(name)

	// Init providers
	var providers []Provider
	for _, providerCfg := range config.Providers {
		provider, err := Registry.BuildProvider(bus, providerCfg)
		if err != nil {
			return nil, errors.Wrap(err, "error in autodiscover provider settings")
		}
		logp.Debug(debugK, "Configured autodiscover provider: %s", provider)
		providers = append(providers, provider)
	}

	return &Autodiscover{
		bus:             bus,
		defaultPipeline: pipeline,
		adapter:         adapter,
		configs:         map[string]map[uint64]*reload.ConfigWithMeta{},
		runners:         cfgfile.NewRunnerList("autodiscover", adapter, pipeline),
		providers:       providers,
		meta:            meta.NewMap(),
	}, nil
}

// Start autodiscover process
func (a *Autodiscover) Start() {
	if a == nil {
		return
	}

	logp.Info("Starting autodiscover manager")
	a.listener = a.bus.Subscribe(a.adapter.EventFilter()...)

	// It is important to start the worker first before starting the producer.
	// In hosts that have large number of workloads, it is easy to have an initial
	// sync of workloads to have a count that is greater than 100 (which is the size
	// of the bounded Go channel. Starting the providers before the consumer would
	// result in the channel filling up and never allowing the worker to start up.
	go a.worker()

	for _, provider := range a.providers {
		provider.Start()
	}
}

func (a *Autodiscover) worker() {
	var updated, retry bool

	for {
		select {
		case event := <-a.listener.Events():
			// This will happen on Stop:
			if event == nil {
				return
			}

			if _, ok := event["start"]; ok {
				updated = a.handleStart(event)
			}
			if _, ok := event["stop"]; ok {
				updated = a.handleStop(event)
			}

		case <-time.After(retryPeriod):
		}

		if updated || retry {
			if retry {
				logp.Debug(debugK, "Reloading existing autodiscover configs after error")
			}

			configs := []*reload.ConfigWithMeta{}
			for _, list := range a.configs {
				for _, c := range list {
					configs = append(configs, c)
				}
			}

			err := a.runners.Reload(configs)

			// On error, make sure the next run also updates because some runners were not properly loaded
			retry = err != nil
			// reset updated status
			updated = false
		}
	}
}

func (a *Autodiscover) handleStart(event bus.Event) bool {
	var updated bool

	logp.Debug(debugK, "Got a start event: %v", event)

	eventID := getID(event)
	if eventID == "" {
		logp.Err("Event didn't provide instance id: %+v, ignoring it", event)
		return false
	}

	// Ensure configs list exists for this instance
	if _, ok := a.configs[eventID]; !ok {
		a.configs[eventID] = map[uint64]*reload.ConfigWithMeta{}
	}

	configs, err := a.adapter.CreateConfig(event)
	if err != nil {
		logp.Debug(debugK, "Could not generate config from event %v: %v", event, err)
		return false
	}
	logp.Debug(debugK, "Generated configs: %+v", configs)

	meta := getMeta(event)
	for _, config := range configs {
		hash, err := cfgfile.HashConfig(config)
		if err != nil {
			logp.Debug(debugK, "Could not hash config %v: %v", config, err)
			continue
		}

		err = a.adapter.CheckConfig(config)
		if err != nil {
			logp.Debug(debugK, "Check failed for config %v: %v, won't start runner", config, err)
			continue
		}

		// Update meta no matter what
		dynFields := a.meta.Store(hash, meta)

		if a.configs[eventID][hash] != nil {
			logp.Debug(debugK, "Config %v is already running", config)
			continue
		}

		a.configs[eventID][hash] = &reload.ConfigWithMeta{
			Config: config,
			Meta:   &dynFields,
		}
		updated = true
	}

	return updated
}

func (a *Autodiscover) handleStop(event bus.Event) bool {
	var updated bool

	logp.Debug(debugK, "Got a stop event: %v", event)
	eventID := getID(event)
	if eventID == "" {
		logp.Err("Event didn't provide instance id: %+v, ignoring it", event)
		return false
	}

	if len(a.configs[eventID]) > 0 {
		logp.Debug(debugK, "Stopping %d configs", len(a.configs[eventID]))
		updated = true
	}

	delete(a.configs, eventID)

	return updated
}

func getMeta(event bus.Event) common.MapStr {
	m := event["meta"]
	if m == nil {
		return nil
	}

	logp.Debug(debugK, "Got a meta field in the event")
	meta, ok := m.(common.MapStr)
	if !ok {
		logp.Err("Got a wrong meta field for event %v", event)
		return nil
	}
	return meta
}

// getID returns the event "id" field string if present
func getID(e bus.Event) string {
	provider, ok := e["provider"]
	if !ok {
		return ""
	}

	id, ok := e["id"]
	if !ok {
		return ""
	}

	return fmt.Sprintf("%s:%s", provider, id)
}

// Stop autodiscover process
func (a *Autodiscover) Stop() {
	if a == nil {
		return
	}

	// Stop listening for events
	a.listener.Stop()

	// Stop providers
	for _, provider := range a.providers {
		provider.Stop()
	}

	// Stop runners
	a.runners.Stop()
	logp.Info("Stopped autodiscover manager")
}
