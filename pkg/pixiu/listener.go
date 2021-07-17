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
	"github.com/apache/dubbo-go-pixiu/pkg/filter/header"
)

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

import (
	fc "github.com/dubbogo/dubbo-go-pixiu-filter/pkg/api/config"
	"github.com/dubbogo/dubbo-go-pixiu-filter/pkg/router"
	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go-pixiu/pkg/common/constant"
	"github.com/apache/dubbo-go-pixiu/pkg/common/extension"
	"github.com/apache/dubbo-go-pixiu/pkg/config"
	ctx "github.com/apache/dubbo-go-pixiu/pkg/context"
	h "github.com/apache/dubbo-go-pixiu/pkg/context/http"
	"github.com/apache/dubbo-go-pixiu/pkg/filter/host"
	"github.com/apache/dubbo-go-pixiu/pkg/logger"
	"github.com/apache/dubbo-go-pixiu/pkg/model"
)

// ListenerService the facade of a listener
type ListenerService struct {
	*model.Listener
}

// Start start the listener
func (l *ListenerService) Start() {
	switch l.Address.SocketAddress.Protocol {
	case model.HTTP:
		l.httpListener()
	default:
		panic("unsupported protocol start: " + l.Address.SocketAddress.ProtocolStr)
	}
}

func (l *ListenerService) httpListener() {
	hl := NewDefaultHttpListener()
	hl.pool.New = func() interface{} {
		return l.allocateContext()
	}

	// user customize http config
	var hc model.HttpConfig
	if l.Config != nil {
		if c, ok := l.Config.(*model.HttpConfig); ok {
			hc = *c
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", hl.ServeHTTP)

	srv := http.Server{
		Addr:           resolveAddress(l.Address.SocketAddress.Address + ":" + strconv.Itoa(l.Address.SocketAddress.Port)),
		Handler:        mux,
		ReadTimeout:    resolveStr2Time(hc.ReadTimeoutStr, 20*time.Second),
		WriteTimeout:   resolveStr2Time(hc.WriteTimeoutStr, 20*time.Second),
		IdleTimeout:    resolveStr2Time(hc.IdleTimeoutStr, 20*time.Second),
		MaxHeaderBytes: resolveInt2IntProp(hc.MaxHeaderBytes, 1<<20),
	}

	logger.Infof("[dubbo-go-pixiu] httpListener start at : %s", srv.Addr)

	log.Println(srv.ListenAndServe())
}

func (l *ListenerService) allocateContext() *h.HttpContext {
	return &h.HttpContext{
		Listener:              l.Listener,
		FilterChains:          l.FilterChains,
		HttpConnectionManager: l.findHttpManager(),
		BaseContext:           ctx.NewBaseContext(),
	}
}

func (l *ListenerService) findHttpManager() model.HttpConnectionManager {
	for _, fc := range l.FilterChains {
		for _, f := range fc.Filters {
			if f.Name == constant.HTTPConnectManagerFilter {
				return f.Config.(model.HttpConnectionManager)
			}
		}
	}

	return *DefaultHttpConnectionManager()
}

// DefaultHttpListener
type DefaultHttpListener struct {
	pool sync.Pool
}

// NewDefaultHttpListener create http listener
func NewDefaultHttpListener() *DefaultHttpListener {
	return &DefaultHttpListener{
		pool: sync.Pool{},
	}
}

// ServeHTTP http request entrance.
func (s *DefaultHttpListener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hc := s.pool.Get().(*h.HttpContext)
	hc.Request = r
	hc.ResetWritermen(w)
	hc.Reset()

	api, err := s.routeRequest(hc, r)
	if err != nil {
		s.pool.Put(hc)
		return
	}

	hc.Ctx = context.Background()
	addFilter(hc, api)

	s.handleHTTPRequest(hc)

	s.pool.Put(hc)
}

func addFilter(ctx *h.HttpContext, api router.API) {
	ctx.AppendFilterFunc(extension.GetMustFilterFunc(constant.MetricFilter),
		extension.GetMustFilterFunc(constant.RecoveryFilter), extension.GetMustFilterFunc(constant.TimeoutFilter))
	alc := config.GetBootstrap().StaticResources.AccessLogConfig
	if alc.Enable {
		ctx.AppendFilterFunc(extension.GetMustFilterFunc(constant.AccessLogFilter))
	}
	ctx.AppendFilterFunc(extension.GetMustFilterFunc(constant.RateLimitFilter))

	switch api.Method.IntegrationRequest.RequestType {
	// TODO add some basic filter for diff protocol
	case fc.DubboRequest:

	case fc.HTTPRequest:
		httpFilter(ctx, api.Method.IntegrationRequest)
	}

	ctx.AppendFilterFunc(header.New().Do(), extension.GetMustFilterFunc(constant.RemoteCallFilter))
	ctx.BuildFilters()

	ctx.AppendFilterFunc(extension.GetMustFilterFunc(constant.ResponseFilter))
}

// try to create filter from config.
func httpFilter(ctx *h.HttpContext, request fc.IntegrationRequest) {
	if len(request.Host) != 0 {
		ctx.AppendFilterFunc(host.New(request.Host).Do())
	}
}

func (s *DefaultHttpListener) routeRequest(ctx *h.HttpContext, req *http.Request) (router.API, error) {
	apiDiscSrv := extension.GetMustAPIDiscoveryService(constant.LocalMemoryApiDiscoveryService)
	api, err := apiDiscSrv.GetAPI(req.URL.Path, fc.HTTPVerb(req.Method))
	if err != nil {
		ctx.WriteWithStatus(http.StatusNotFound, constant.Default404Body)
		ctx.AddHeader(constant.HeaderKeyContextType, constant.HeaderValueTextPlain)
		e := errors.Errorf("Requested URL %s not found", req.URL.Path)
		logger.Debug(e.Error())
		return router.API{}, e
	}
	if !api.Method.OnAir {
		ctx.WriteWithStatus(http.StatusNotAcceptable, constant.Default406Body)
		ctx.AddHeader(constant.HeaderKeyContextType, constant.HeaderValueTextPlain)
		e := errors.Errorf("Requested API %s %s does not online", req.Method, req.URL.Path)
		logger.Debug(e.Error())
		return router.API{}, e
	}
	ctx.API(api)
	return api, nil
}

func (s *DefaultHttpListener) handleHTTPRequest(c *h.HttpContext) {
	if len(c.BaseContext.Filters) > 0 {
		c.Next()
		c.WriteHeaderNow()
		return
	}

	// TODO redirect
}

func resolveInt2IntProp(currentV, defaultV int) int {
	if currentV == 0 {
		return defaultV
	}

	return currentV
}

func resolveStr2Time(currentV string, defaultV time.Duration) time.Duration {
	if currentV == "" {
		return defaultV
	} else {
		if duration, err := time.ParseDuration(currentV); err != nil {
			return 20 * time.Second
		} else {
			return duration
		}
	}
}

func resolveAddress(addr string) string {
	if addr == "" {
		logger.Debug("Addr is undefined. Using port :8080 by default")
		return ":8080"
	}

	return addr
}
