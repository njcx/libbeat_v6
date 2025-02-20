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

package add_cloud_metadata

import (
	"github.com/njcx/libbeat_v6/common"
	s "github.com/njcx/libbeat_v6/common/schema"
	c "github.com/njcx/libbeat_v6/common/schema/mapstriface"
)

// AWS EC2 Metadata Service
func newEc2MetadataFetcher(config *common.Config) (*metadataFetcher, error) {
	ec2InstanceIdentityURI := "/2014-02-25/dynamic/instance-identity/document"
	ec2Schema := func(m map[string]interface{}) common.MapStr {
		out, _ := s.Schema{
			"instance_id":       c.Str("instanceId"),
			"machine_type":      c.Str("instanceType"),
			"region":            c.Str("region"),
			"availability_zone": c.Str("availabilityZone"),
		}.Apply(m)
		return out
	}

	fetcher, err := newMetadataFetcher(config, "ec2", nil, metadataHost, ec2Schema, ec2InstanceIdentityURI)
	return fetcher, err
}
