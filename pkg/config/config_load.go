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
	"io/ioutil"
	"log"
	"path/filepath"
)

import (
	"github.com/ghodss/yaml"
	"github.com/goinggo/mapstructure"
)

import (
	"github.com/apache/dubbo-go-pixiu/pkg/common/constant"
	"github.com/apache/dubbo-go-pixiu/pkg/logger"
	"github.com/apache/dubbo-go-pixiu/pkg/model"
)

var (
	configPath     string
	config         *model.Bootstrap
	configLoadFunc LoadFunc = LoadYAMLConfig
)

// LoadFunc ConfigLoadFunc parse a input(usually file path) into a pixiu config
type LoadFunc func(path string) *model.Bootstrap

// GetBootstrap get config global, need a better name
func GetBootstrap() *model.Bootstrap {
	return config
}

// Load config file and parse
func Load(path string) *model.Bootstrap {
	logger.Infof("[dubbopixiu go] load path:%s", path)
	configPath, _ = filepath.Abs(path)
	if configPath != "" && CheckYamlFormat(configPath) {
		RegisterConfigLoadFunc(LoadYAMLConfig)
	}
	if cfg := configLoadFunc(configPath); cfg != nil {
		config = cfg
	}
	return config
}

// RegisterConfigLoadFunc can replace a new config load function instead of default
func RegisterConfigLoadFunc(f LoadFunc) {
	configLoadFunc = f
}

func CheckYamlFormat(path string) bool {
	ext := filepath.Ext(path)
	if ext == constant.YAML || ext == constant.YML {
		return true
	}
	return false
}

// LoadYAMLConfig YAMLConfigLoad config load yaml
func LoadYAMLConfig(path string) *model.Bootstrap {
	log.Println("load config in YAML format from : ", path)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalln("[config] [yaml load] load config failed, ", err)
	}
	cfg := &model.Bootstrap{}
	err = yaml.Unmarshal(content, cfg)
	if err != nil {
		log.Fatalln("[config] [yaml load] convert YAML to JSON failed, ", err)
	}
	err = Adapter(cfg)
	if err != nil {
		log.Fatalln("[config] [yaml load] yaml unmarshal config failed, ", err)
	}
	return cfg
}

func Adapter(cfg *model.Bootstrap) (err error) {
	if GetFilterChain(cfg) != nil || GetHttpConfig(cfg) != nil || GetProtocol(cfg) != nil ||
		GetLoadBalance(cfg) != nil || GetDiscoveryType(cfg) != nil {
		return err
	}
	return nil
}

func GetProtocol(cfg *model.Bootstrap) (err error) {
	if cfg == nil {
		logger.Error("Bootstrap configuration is null")
		return err
	}
	for _, l := range cfg.StaticResources.Listeners {
		if l.Address.SocketAddress.ProtocolStr == "" {
			l.Address.SocketAddress.ProtocolStr = constant.DefaultProtocolType
		}
		l.Address.SocketAddress.Protocol = model.ProtocolType(model.ProtocolTypeValue[l.Address.SocketAddress.ProtocolStr])
	}
	return nil
}

func GetHttpConfig(cfg *model.Bootstrap) (err error) {
	if cfg == nil {
		logger.Error("Bootstrap configuration is null")
		return err
	}
	for i, l := range cfg.StaticResources.Listeners {
		hc := &model.HttpConfig{}
		if l.Config != nil {
			if v, ok := l.Config.(map[string]interface{}); ok {
				logger.Info("http config:", v, ok)
				switch l.Name {
				case constant.DefaultHTTPType:
					if err := mapstructure.Decode(v, hc); err != nil {
						logger.Error(err)
					}
					cfg.StaticResources.Listeners[i].Config = *hc
				}
			}
		}
	}
	return nil
}

func GetFilterChain(cfg *model.Bootstrap) (err error) {
	if cfg == nil {
		logger.Error("Bootstrap configuration is null")
		return err
	}
	for _, l := range cfg.StaticResources.Listeners {
		for _, fc := range l.FilterChains {
			if fc.Filters != nil {
				for i, fcf := range fc.Filters {
					hcm := &model.HttpConnectionManager{}
					if fcf.Config != nil {
						switch fcf.Name {
						case constant.DefaultFilterType:
							if v, ok := fcf.Config.(map[string]interface{}); ok {
								if err := mapstructure.Decode(v, hcm); err != nil {
									logger.Error(err)
								}
								fc.Filters[i].Config = *hcm
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func GetLoadBalance(cfg *model.Bootstrap) (err error) {
	if cfg == nil {
		logger.Error("Bootstrap configuration is null")
		return err
	}
	var lbPolicy int32
	for _, c := range cfg.StaticResources.Clusters {
		flag := true
		if c.TypeStr != "" {
			if t, ok := model.LbPolicyValue[c.LbStr]; ok {
				lbPolicy = t
				flag = false
			}
		}
		if flag {
			c.LbStr = constant.DefaultLoadBalanceType
			lbPolicy = model.LbPolicyValue[c.LbStr]
		}
		c.Type = model.DiscoveryType(lbPolicy)
	}
	return nil
}

func GetDiscoveryType(cfg *model.Bootstrap) (err error) {
	if cfg == nil {
		logger.Error("Bootstrap configuration is null")
		return err
	}
	var discoveryType int32
	for _, c := range cfg.StaticResources.Clusters {
		flag := true
		if c.TypeStr != "" {
			if t, ok := model.DiscoveryTypeValue[c.TypeStr]; ok {
				discoveryType = t
				flag = false
			}
		}
		if flag {
			c.TypeStr = constant.DefaultDiscoveryType
			discoveryType = model.DiscoveryTypeValue[c.TypeStr]
		}
		c.Type = model.DiscoveryType(discoveryType)
	}
	return nil
}
