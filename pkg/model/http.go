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

package model

import (
	"github.com/apache/dubbo-go-pixiu/pkg/logger"
	"github.com/mitchellh/mapstructure"
)

// HttpConnectionManager
type HttpConnectionManager struct {
	RouteConfig       RouteConfiguration     `yaml:"route_config" json:"route_config" mapstructure:"route_config"`
	AuthorityConfig   AuthorityConfiguration `yaml:"authority_config" json:"authority_config" mapstructure:"authority_config"`
	HTTPFilters       []HTTPFilter           `yaml:"http_filters" json:"http_filters" mapstructure:"http_filters"`
	ServerName        string                 `yaml:"server_name" json:"server_name" mapstructure:"server_name"`
	IdleTimeoutStr    string                 `yaml:"idle_timeout" json:"idle_timeout" mapstructure:"idle_timeout"`
	GenerateRequestID bool                   `yaml:"generate_request_id" json:"generate_request_id" mapstructure:"generate_request_id"`
}

// CorsPolicy
type CorsPolicy struct {
	AllowOrigin      []string `yaml:"allow_origin" json:"allow_origin" mapstructure:"allow_origin"`
	AllowMethods     string   // access-control-allow-methods
	AllowHeaders     string   // access-control-allow-headers
	ExposeHeaders    string   // access-control-expose-headers
	MaxAge           string   // access-control-max-age
	AllowCredentials bool
	Enabled          bool `yaml:"enabled" json:"enabled" mapstructure:"enabled"`
}

// HTTPFilter http filter
type HTTPFilter struct {
	Name   string      `yaml:"name" json:"name" mapstructure:"name"`
	Config interface{} `yaml:"config" json:"config" mapstructure:"config"`
}

type RequestMethod int32

const (
	METHOD_UNSPECIFIED = 0 + iota // (DEFAULT)
	GET
	HEAD
	POST
	PUT
	DELETE
	CONNECT
	OPTIONS
	TRACE
)

var RequestMethodName = map[int32]string{
	0: "METHOD_UNSPECIFIED",
	1: "GET",
	2: "HEAD",
	3: "POST",
	4: "PUT",
	5: "DELETE",
	6: "CONNECT",
	7: "OPTIONS",
	8: "TRACE",
}

var RequestMethodValue = map[string]int32{
	"METHOD_UNSPECIFIED": 0,
	"GET":                1,
	"HEAD":               2,
	"POST":               3,
	"PUT":                4,
	"DELETE":             5,
	"CONNECT":            6,
	"OPTIONS":            7,
	"TRACE":              8,
}

// HttpConfig the http config
type HttpConfig struct {
	IdleTimeoutStr  string `yaml:"idle_timeout" json:"idle_timeout" mapstructure:"idle_timeout"`
	ReadTimeoutStr  string `json:"read_timeout,omitempty" yaml:"read_timeout,omitempty" mapstructure:"read_timeout"`
	WriteTimeoutStr string `json:"write_timeout,omitempty" yaml:"write_timeout,omitempty" mapstructure:"write_timeout"`
	MaxHeaderBytes  int    `json:"max_header_bytes,omitempty" yaml:"max_header_bytes,omitempty" mapstructure:"max_header_bytes"`
	CertFile        string `yaml:"cert_file" json:"cert_file" mapstructure:"cert_file"`
	KeyFile         string `yaml:"key_file" json:"key_file" mapstructure:"key_file"`
}

func MapInStruct(cfg interface{}) *HttpConfig {
	var hc *HttpConfig
	if cfg != nil {
		if ok := mapstructure.Decode(cfg, &hc); ok != nil {
			logger.Error("Config error", ok)
		}
	}
	return hc
}
