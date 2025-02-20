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

package actions

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/njcx/libbeat_v6/beat"
	"github.com/njcx/libbeat_v6/common"
	"github.com/njcx/libbeat_v6/logp"
	"github.com/njcx/libbeat_v6/processors"
)

type renameFields struct {
	config renameFieldsConfig
}

type renameFieldsConfig struct {
	Fields        []fromTo `config:"fields"`
	IgnoreMissing bool     `config:"ignore_missing"`
	FailOnError   bool     `config:"fail_on_error"`
}

type fromTo struct {
	From string `config:"from"`
	To   string `config:"to"`
}

func init() {
	processors.RegisterPlugin("rename",
		configChecked(newRenameFields,
			requireFields("fields")))
}

func newRenameFields(c *common.Config) (processors.Processor, error) {
	config := renameFieldsConfig{
		IgnoreMissing: false,
		FailOnError:   true,
	}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack the rename configuration: %s", err)
	}

	f := &renameFields{
		config: config,
	}
	return f, nil
}

func (f *renameFields) Run(event *beat.Event) (*beat.Event, error) {
	var backup common.MapStr
	// Creates a copy of the event to revert in case of failure
	if f.config.FailOnError {
		backup = event.Fields.Clone()
	}

	for _, field := range f.config.Fields {
		err := f.renameField(field.From, field.To, event.Fields)
		if err != nil && f.config.FailOnError {
			logp.Debug("rename", "Failed to rename fields, revert to old event: %s", err)
			event.Fields = backup
			return event, err
		}
	}

	return event, nil
}

func (f *renameFields) renameField(from string, to string, fields common.MapStr) error {
	// Fields cannot be overwritten. Either the target field has to be dropped first or renamed first
	exists, _ := fields.HasKey(to)
	if exists {
		return fmt.Errorf("target field %s already exists, drop or rename this field first", to)
	}

	value, err := fields.GetValue(from)
	if err != nil {
		// Ignore ErrKeyNotFound errors
		if f.config.IgnoreMissing && errors.Cause(err) == common.ErrKeyNotFound {
			return nil
		}
		return fmt.Errorf("could not fetch value for key: %s, Error: %s", from, err)
	}

	// Deletion must happen first to support cases where a becomes a.b
	err = fields.Delete(from)
	if err != nil {
		return fmt.Errorf("could not delete key: %s,  %+v", from, err)
	}

	_, err = fields.Put(to, value)
	if err != nil {
		return fmt.Errorf("could not put value: %s: %v, %+v", to, value, err)
	}
	return nil
}

func (f *renameFields) String() string {
	return "rename=" + fmt.Sprintf("%+v", f.config.Fields)
}
