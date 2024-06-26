/*-
 * Copyright (c) 2021, F5 Networks, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package controller

import (
	"github.com/F5Networks/f5-ipam-controller/pkg/ipamspec"
	"github.com/F5Networks/f5-ipam-controller/pkg/manager"
	"github.com/F5Networks/f5-ipam-controller/pkg/orchestration"
	log "github.com/F5Networks/f5-ipam-controller/pkg/vlogger"
)

type Spec struct {
	Orchestrator orchestration.Orchestrator
	Manager      manager.Manager
	StopCh       chan struct{}
}

type Controller struct {
	Spec
	reqChan  chan ipamspec.IPAMRequest
	respChan chan ipamspec.IPAMResponse
}

func NewController(spec Spec) *Controller {
	ctlr := &Controller{
		Spec:     spec,
		reqChan:  make(chan ipamspec.IPAMRequest),
		respChan: make(chan ipamspec.IPAMResponse),
	}

	return ctlr
}

func (ctlr *Controller) runController() {
	for req := range ctlr.reqChan {
		switch req.Operation {
		case ipamspec.CREATE:

			sendResponse := func(request ipamspec.IPAMRequest, ipAddr string) {
				resp := ipamspec.IPAMResponse{
					Request: request,
					IPAddr:  ipAddr,
					Status:  true,
				}
				ctlr.respChan <- resp
			}

			ipAddr := ctlr.Manager.GetIPAddress(req)
			if ipAddr != "" {
				go sendResponse(req, ipAddr)
				break
			}

			ipAddr = ctlr.Manager.AllocateNextIPAddress(req)
			if ipAddr != "" {
				log.Debugf("[CORE] Allocated IP: %v for Request: %v", ipAddr, req.String())
				// A Record Support is disabled
				//req.IPAddr = ipAddr
				//ok := ctlr.Manager.CreateARecord(req)
				//if !ok {
				//	req.IPAddr = ipAddr
				//	ctlr.Manager.ReleaseIPAddress(req)
				//	log.Errorf("[CORE] Unable to Create A Record with hostname: %v", req.HostName)
				//	log.Infof("[CORE] Releasing Allocated IP: %v", ipAddr)
				//
				//	break
				//}
				go sendResponse(req, ipAddr)
			}
		case ipamspec.DELETE:
			ipAddr := ctlr.Manager.GetIPAddress(req)
			if ipAddr != "" {
				req.IPAddr = ipAddr
				ctlr.Manager.ReleaseIPAddress(req)
				// A Record Support is disabled
				//ctlr.Manager.DeleteARecord(req)
			}
			go func(request ipamspec.IPAMRequest) {
				resp := ipamspec.IPAMResponse{
					Request: request,
					IPAddr:  "",
					Status:  true,
				}
				ctlr.respChan <- resp
			}(req)
		}
	}
}

func (ctlr *Controller) Start() {
	ctlr.Orchestrator.SetupCommunicationChannels(
		ctlr.reqChan,
		ctlr.respChan,
	)
	log.Info("[CORE] Controller started")

	ctlr.Orchestrator.Start(ctlr.StopCh)

	go ctlr.runController()
}

func (ctlr *Controller) Stop() {
	ctlr.Orchestrator.Stop()
}
