/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package pixiu

import (
	"net/http"
	"strconv"
	"sync"
)

import (
	"github.com/apache/dubbo-go-pixiu/pkg/client/dubbo"
	"github.com/apache/dubbo-go-pixiu/pkg/common/constant"
	"github.com/apache/dubbo-go-pixiu/pkg/config"
	// The filter needs to be initialized
	_ "github.com/apache/dubbo-go-pixiu/pkg/filter"
	"github.com/apache/dubbo-go-pixiu/pkg/initialize"
	"github.com/apache/dubbo-go-pixiu/pkg/logger"
	"github.com/apache/dubbo-go-pixiu/pkg/model"
	"github.com/apache/dubbo-go-pixiu/pkg/service/api"
)

// PX is Pixiu start struct
type PX struct {
	startWG sync.WaitGroup
}

// Start pixiu start
func (p *PX) Start() {
	conf := config.GetBootstrap()

	p.startWG.Add(1)

	defer func() {
		if re := recover(); re != nil {
			logger.Error(re)
			// TODO stop
		}
	}()

	p.beforeStart()

	registerOtelMetricMeter(conf.Metric)

	listeners := conf.GetListeners()

	for _, s := range listeners {
		ls := ListenerService{Listener: s}
		go ls.Start()
	}

	if conf.GetPprof().Enable {
		addr := conf.GetPprof().Address.SocketAddress
		if len(addr.Address) == 0 {
			addr.Address = constant.PprofDefaultAddress
		}
		if addr.Port == 0 {
			addr.Port = constant.PprofDefaultPort
		}
		go http.ListenAndServe(addr.Address+":"+strconv.Itoa(addr.Port), nil)
		logger.Infof("[dubbopixiu go pprof] httpListener start by : %s", addr.Address+":"+strconv.Itoa(addr.Port))
	}
}

func (p *PX) beforeStart() {
	dubbo.SingletonDubboClient().Init()
	initialize.Run(config.GetAPIConf())
	if err := api.InitAPIsFromConfig(config.GetAPIConf()); err != nil {
		logger.Errorf("InitAPIsFromConfig fail: %v", err)
	}
}

// NewPX create pixiu
func NewPX() *PX {
	return &PX{
		startWG: sync.WaitGroup{},
	}
}

func Start(bs *model.Bootstrap) {
	logger.Infof("[dubbopixiu go] start by config : %+v", bs)

	proxy := NewPX()
	proxy.Start()

	proxy.startWG.Wait()
}
