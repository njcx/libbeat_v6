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

// Azure VM Metadata Service
func newAzureVmMetadataFetcher(config *common.Config) (*metadataFetcher, error) {
	azMetadataURI := "/metadata/instance/compute?api-version=2017-04-02"
	azHeaders := map[string]string{"Metadata": "true"}
	azSchema := func(m map[string]interface{}) common.MapStr {
		out, _ := s.Schema{
			"instance_id":   c.Str("vmId"),
			"instance_name": c.Str("name"),
			"machine_type":  c.Str("vmSize"),
			"region":        c.Str("location"),
		}.Apply(m)
		return out
	}

	fetcher, err := newMetadataFetcher(config, "az", azHeaders, metadataHost, azSchema, azMetadataURI)
	return fetcher, err
}
