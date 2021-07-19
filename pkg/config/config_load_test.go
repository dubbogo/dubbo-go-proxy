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

package config

import (
	"encoding/json"
	"log"
	"os"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go-pixiu/pkg/model"
)

var b model.Bootstrap

func TestMain(m *testing.M) {
	log.Println("Prepare Bootstrap")
	b = model.Bootstrap{
		StaticResources: model.StaticResources{
			Listeners: []*model.Listener{
				{
					Name: "net/http",
					Address: model.Address{
						SocketAddress: model.SocketAddress{
							ProtocolStr: "HTTPS",
							Address:     "0.0.0.0",
							Port:        443,
						},
					},
					Config: model.HttpConfig{
						IdleTimeoutStr:  "5s",
						WriteTimeoutStr: "5s",
						ReadTimeoutStr:  "5s",
					},
					FilterChains: []model.FilterChain{
						{
							FilterChainMatch: model.FilterChainMatch{
								Domains: []string{
									"api.dubbo.com",
									"api.pixiu.com",
								},
							},
							Filters: []model.Filter{
								{
									Name: "dgp.filters.http_connect_manager",
									Config: model.HttpConnectionManager{
										RouteConfig: model.RouteConfiguration{
											Routes: []model.Router{
												{
													Match: model.RouterMatch{
														Prefix: "/api/v1",
														Headers: []model.HeaderMatcher{
															{Name: "X-DGP-WAY",
																Value: "dubbo",
															},
														},
													},
													Route: model.RouteAction{
														Cluster:                     "test_dubbo",
														ClusterNotFoundResponseCode: 505,
														Cors: model.CorsPolicy{
															AllowOrigin: []string{
																"*",
															},
															Enabled: true,
														},
													},
												},
											},
										},
										HTTPFilters: []model.HTTPFilter{
											{
												Name:   "dgp.filters.http.api",
												Config: interface{}(nil),
											},
											{
												Name:   "dgp.filters.http.router",
												Config: interface{}(nil),
											},
											{
												Name:   "dgp.filters.http_transfer_dubbo",
												Config: interface{}(nil),
											},
										},
										ServerName:        "test_http_dubbo",
										GenerateRequestID: false,
									},
								},
							},
						},
					},
				},
			},
			Clusters: []*model.Cluster{
				{
					Name:    "test_dubbo",
					TypeStr: "EDS",
					Type:    model.EDS,
					LbStr:   "RoundRobin",
					Registries: map[string]model.Registry{
						"zookeeper": {
							Timeout:  "3s",
							Address:  "127.0.0.1:2182",
							Username: "",
							Password: "",
						},
						"consul": {
							Timeout: "3s",
							Address: "127.0.0.1:8500",
						},
					},
				},
			},
			TimeoutConfig: model.TimeoutConfig{
				ConnectTimeoutStr: "5s",
				RequestTimeoutStr: "10s",
			},
			ShutdownConfig: &model.ShutdownConfig{
				Timeout:      "60s",
				StepTimeout:  "10s",
				RejectPolicy: "immediacy",
			},
			PprofConf: model.PprofConf{
				Enable: true,
				Address: model.Address{
					SocketAddress: model.SocketAddress{
						Address: "0.0.0.0",
						Port:    6060,
					},
				},
			},
		},
	}
	retCode := m.Run()
	os.Exit(retCode)
}

func TestLoad(t *testing.T) {
	conf := Load("conf_test.yaml")
	assert.Equal(t, 1, len(conf.StaticResources.Listeners))
	assert.Equal(t, 1, len(conf.StaticResources.Clusters))
	Adapter(&b)
	assert.Equal(t, *conf, b)
}

func TestStruct2JSON(t *testing.T) {
	if bytes, err := json.Marshal(b); err != nil {
		t.Fatal(err)
	} else {
		t.Log(string(bytes))
	}
}
