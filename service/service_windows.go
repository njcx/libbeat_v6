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

package service

import (
	"os"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"

	"github.com/njcx/libbeat_v6/logp"
)

type beatService struct{}

// Execute runs the beat service with the arguments and manages changes that
// occur in the environment or runtime that may affect the beat.
func (m *beatService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for c := range r {
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
			// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
			time.Sleep(100 * time.Millisecond)
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			break loop
		default:
			logp.Err("Unexpected control request: $%d. Ignored.", c)
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

// ProcessWindowsControlEvents on Windows machines creates a loop
// that only finishes when a Stop or Shutdown request is received.
// On non-windows platforms, the function does nothing. The
// stopCallback function is called when the Stop/Shutdown
// request is received.
func ProcessWindowsControlEvents(stopCallback func()) {
	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		logp.Err("IsAnInteractiveSession: %v", err)
		return
	}
	logp.Debug("service", "Windows is interactive: %v", isInteractive)

	run := svc.Run
	if isInteractive {
		run = debug.Run
	}
	err = run(os.Args[0], &beatService{})
	if err != nil {
		logp.Err("Error: %v", err)
	} else {
		stopCallback()
	}
}
