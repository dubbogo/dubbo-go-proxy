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

// Router struct
type Router struct {
	Match    RouterMatch `yaml:"match" json:"match" mapstructure:"match"`
	Route    RouteAction `yaml:"route" json:"route" mapstructure:"route"`
	Redirect RouteAction `yaml:"redirect" json:"redirect" mapstructure:"redirect"`
	//"metadata": "{...}",
	//"decorator": "{...}"
}

// RouterMatch
type RouterMatch struct {
	Prefix        string          `yaml:"prefix" json:"prefix" mapstructure:"prefix"`
	Path          string          `yaml:"path" json:"path" mapstructure:"path"`
	Regex         string          `yaml:"regex" json:"regex" mapstructure:"regex"`
	CaseSensitive bool            // CaseSensitive default true
	Headers       []HeaderMatcher `yaml:"headers" json:"headers" mapstructure:"headers"`
}

// RouteAction match route should do
type RouteAction struct {
	Cluster                     string            `yaml:"cluster" json:"cluster" mapstructure:"cluster"`
	ClusterNotFoundResponseCode int               `yaml:"cluster_not_found_response_code" json:"cluster_not_found_response_code" mapstructure:"cluster_not_found_response_code"`
	PrefixRewrite               string            `yaml:"prefix_rewrite" json:"prefix_rewrite" mapstructure:"prefix_rewrite"`
	HostRewrite                 string            `yaml:"host_rewrite" json:"host_rewrite" mapstructure:"host_rewrite"`
	Timeout                     string            `yaml:"timeout" json:"timeout" mapstructure:"timeout"`
	Priority                    int8              `yaml:"priority" json:"priority" mapstructure:"priority"`
	ResponseHeadersToAdd        HeaderValueOption `yaml:"response_headers_to_add" json:"response_headers_to_add" mapstructure:"response_headers_to_add"`          // ResponseHeadersToAdd add response head
	ResponseHeadersToRemove     []string          `yaml:"response_headers_to_remove" json:"response_headers_to_remove" mapstructure:"response_headers_to_remove"` // ResponseHeadersToRemove remove response head
	RequestHeadersToAdd         HeaderValueOption `yaml:"request_headers_to_add" json:"request_headers_to_add" mapstructure:"request_headers_to_add"`             // RequestHeadersToAdd add request head
	Cors                        CorsPolicy        `yaml:"cors" json:"cors" mapstructure:"cors"`
}

// RouteConfiguration
type RouteConfiguration struct {
	InternalOnlyHeaders     []string          `yaml:"internal_only_headers" json:"internal_only_headers" mapstructure:"internal_only_headers"`                // InternalOnlyHeaders used internal, clear http request head
	ResponseHeadersToAdd    HeaderValueOption `yaml:"response_headers_to_add" json:"response_headers_to_add" mapstructure:"response_headers_to_add"`          // ResponseHeadersToAdd add response head
	ResponseHeadersToRemove []string          `yaml:"response_headers_to_remove" json:"response_headers_to_remove" mapstructure:"response_headers_to_remove"` // ResponseHeadersToRemove remove response head
	RequestHeadersToAdd     HeaderValueOption `yaml:"request_headers_to_add" json:"request_headers_to_add" mapstructure:"request_headers_to_add"`             // RequestHeadersToAdd add request head
	Routes                  []Router          `yaml:"routes" json:"routes" mapstructure:"routes"`
}
