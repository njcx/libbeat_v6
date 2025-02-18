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

package includes

import (
	// import queue types
	_ "github.com/njcx/libbeat_v6/outputs/codec/format"
	_ "github.com/njcx/libbeat_v6/outputs/codec/json"
	_ "github.com/njcx/libbeat_v6/outputs/console"
	_ "github.com/njcx/libbeat_v6/outputs/elasticsearch"
	_ "github.com/njcx/libbeat_v6/outputs/fileout"
	_ "github.com/njcx/libbeat_v6/outputs/kafka"
	_ "github.com/njcx/libbeat_v6/outputs/logstash"
	_ "github.com/njcx/libbeat_v6/outputs/redis"
	_ "github.com/njcx/libbeat_v6/publisher/queue/memqueue"
	_ "github.com/njcx/libbeat_v6/publisher/queue/spool"
)
