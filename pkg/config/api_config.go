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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	fc "github.com/dubbogo/dubbo-go-pixiu-filter/pkg/api/config"
	fr "github.com/dubbogo/dubbo-go-pixiu-filter/pkg/api/config/ratelimit"
	etcdv3 "github.com/dubbogo/gost/database/kv/etcd/v3"
	perrors "github.com/pkg/errors"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
)

import (
	"github.com/apache/dubbo-go-pixiu/pkg/common/yaml"
	"github.com/apache/dubbo-go-pixiu/pkg/logger"
	"github.com/apache/dubbo-go-pixiu/pkg/model"
)

var (
	apiConfig *fc.APIConfig
	once      sync.Once
	client    *etcdv3.Client
	listener  APIConfigResourceListener
	lock      sync.RWMutex
)

var (
	BASE_INFO_NAME = "name"
	BASE_INFO_DESC = "description"
)

// APIConfigResourceListener defines api resource and method config listener interface
type APIConfigResourceListener interface {
	// ResourceChange handle modify resource event
	ResourceChange(new fc.Resource, old fc.Resource) bool // bool is return for interface implement is interesting
	// ResourceAdd handle add resource event
	ResourceAdd(res fc.Resource) bool
	// ResourceDelete handle delete resource event
	ResourceDelete(deleted fc.Resource) bool
	// MethodChange handle modify method event
	MethodChange(res fc.Resource, method fc.Method, old fc.Method) bool
	// MethodAdd handle add method below one resource event
	MethodAdd(res fc.Resource, method fc.Method) bool
	// MethodDelete handle delete method event
	MethodDelete(res fc.Resource, method fc.Method) bool

	RateLimitChange(*fr.Config)
}

// LoadAPIConfigFromFile load the api config from file
func LoadAPIConfigFromFile(path string) (*fc.APIConfig, error) {
	if len(path) == 0 {
		return nil, perrors.Errorf("Config file not specified")
	}
	logger.Infof("Load API configuration file form %s", path)
	apiConf := &fc.APIConfig{}
	err := yaml.UnmarshalYMLConfig(path, apiConf)
	if err != nil {
		return nil, perrors.Errorf("unmarshalYmlConfig error %s", perrors.WithStack(err))
	}
	apiConfig = apiConf
	return apiConf, nil
}

// LoadAPIConfig load the api config from config center
func LoadAPIConfig(metaConfig *model.APIMetaConfig) (*fc.APIConfig, error) {
	tmpClient, err := etcdv3.NewConfigClientWithErr(
		etcdv3.WithName(etcdv3.RegistryETCDV3Client),
		etcdv3.WithTimeout(10*time.Second),
		etcdv3.WithEndpoints(strings.Split(metaConfig.Address, ",")...),
	)
	if err != nil {
		return nil, perrors.Errorf("Init etcd client fail error %s", err)
	}

	client = tmpClient
	kList, vList, err := client.GetChildren(metaConfig.APIConfigPath)
	if err != nil {
		return nil, perrors.Errorf("Get remote config fail error %s", err)
	}
	if err = initAPIConfigFromKVList(kList, vList); err != nil {
		return nil, err
	}
	// TODO: init other setting which need fetch from remote
	go listenResourceAndMethodEvent(metaConfig.APIConfigPath)
	// TODO: watch other setting which need fetch from remote
	return apiConfig, nil
}

func initAPIConfigFromKVList(kList, vList []string) error {
	var skList, svList, mkList, mvList []string
	var baseInfo string
	var rateLimit string

	for i, k := range kList {
		v := vList[i]
		//handle base info
		re := getCheckBaseInfoRegexp()
		if m := re.Match([]byte(k)); m {
			baseInfo = v
			continue
		}

		// handle resource
		re = getCheckResourceRegexp()
		if m := re.Match([]byte(k)); m {
			skList = append(skList, k)
			svList = append(svList, v)
			continue
		}
		// handle method
		re = getExtractMethodRegexp()
		if m := re.Match([]byte(k)); m {
			mkList = append(mkList, k)
			mvList = append(mvList, v)
			continue
		}

		//handle rate limit config
		re = getCheckRatelimitRegexp()
		if m := re.Match([]byte(k)); m {
			rateLimit = v
			continue
		}
	}

	lock.Lock()
	defer lock.Unlock()

	tmpApiConf := &fc.APIConfig{}
	if err := initBaseInfoFromString(tmpApiConf, baseInfo); err != nil {
		logger.Errorf("initBaseInfoFromString error %s", err)
		return err
	}
	if err := initAPIConfigServiceFromKvList(tmpApiConf, skList, svList); err != nil {
		logger.Errorf("initAPIConfigServiceFromKvList error %s", err)
		return err
	}
	if err := initAPIConfigMethodFromKvList(tmpApiConf, mkList, mvList); err != nil {
		logger.Errorf("initAPIConfigMethodFromKvList error %s", err)
		return err
	}
	if err := initAPIConfigRatelimitFromString(tmpApiConf, rateLimit); err != nil {
		logger.Errorf("initAPIConfigRatelimitFromString error %s", err)
		return err
	}

	apiConfig = tmpApiConf
	return nil
}

func initBaseInfoFromString(conf *fc.APIConfig, str string) error {
	properties := make(map[string]string, 8)
	if err := yaml.UnmarshalYML([]byte(str), properties); err != nil {
		logger.Errorf("unmarshalYmlConfig error %s", err)
		return err
	}
	if v, ok := properties[BASE_INFO_NAME]; ok {
		conf.Name = v
	}
	if v, ok := properties[BASE_INFO_DESC]; ok {
		conf.Description = v
	}
	return nil
}

func initAPIConfigRatelimitFromString(conf *fc.APIConfig, str string) error {
	c := fr.Config{}
	if err := yaml.UnmarshalYML([]byte(str), &c); err != nil {
		logger.Errorf("unmarshalYmlConfig error %s", err)
		return err
	}
	conf.RateLimit = c
	return nil
}

func initAPIConfigMethodFromKvList(config *fc.APIConfig, kList, vList []string) error {
	for i := range kList {
		v := vList[i]
		method := &fc.Method{}
		err := yaml.UnmarshalYML([]byte(v), method)
		if err != nil {
			logger.Errorf("unmarshalYmlConfig error %s", err)
			return err
		}

		found := false
		for r, resource := range config.Resources {
			if method.ResourcePath != resource.Path {
				continue
			}

			for j, old := range resource.Methods {
				if old.HTTPVerb == method.HTTPVerb {
					// modify one method
					resource.Methods[j] = *method
					found = true
				}
			}
			if !found {
				resource.Methods = append(resource.Methods, *method)
				config.Resources[r] = resource
				found = true
			}
		}

		// not found one resource, so need add empty resource first
		if !found {
			resource := &fc.Resource{}
			resource.Methods = append(resource.Methods, *method)
			resource.Path = method.ResourcePath
			config.Resources = append(config.Resources, *resource)
		}
	}
	return nil
}

func initAPIConfigServiceFromKvList(config *fc.APIConfig, kList, vList []string) error {
	for i := range kList {
		v := vList[i]
		resource := &fc.Resource{}
		err := yaml.UnmarshalYML([]byte(v), resource)
		if err != nil {
			logger.Errorf("unmarshalYmlConfig error %s", err)
			return err
		}

		found := false
		if config.Resources == nil {
			config.Resources = make([]fc.Resource, 0)
		}

		for i, old := range config.Resources {
			if old.Path != resource.Path {
				continue
			}
			// replace old with new one except method list
			resource.Methods = old.Methods
			config.Resources[i] = *resource
			found = true
		}

		if !found {
			config.Resources = append(config.Resources, *resource)
		}
		continue
	}
	return nil
}

func listenResourceAndMethodEvent(key string) bool {
	for {
		wc, err := client.WatchWithOption(key, clientv3.WithPrefix())
		if err != nil {
			logger.Warnf("Watch api config {key:%s} = error{%s}", key, err)
			return false
		}

		select {

		// client stopped
		case <-client.Done():
			logger.Warnf("client stopped")
			return false
		// client ctx stop
		// handle etcd events
		case e, ok := <-wc:
			if !ok {
				logger.Warnf("watch-chan closed")
				return false
			}

			if e.Err() != nil {
				logger.Errorf("watch ERR {err: %s}", e.Err())
				continue
			}
			for _, event := range e.Events {
				switch event.Type {
				case mvccpb.PUT:
					logger.Infof("get event (key{%s}) = event{EventNodePut}", event.Kv.Key)
					handlePutEvent(event.Kv.Key, event.Kv.Value)
				case mvccpb.DELETE:
					logger.Infof("get event (key{%s}) = event{EventNodeDeleted}", event.Kv.Key)
					handleDeleteEvent(event.Kv.Key, event.Kv.Value)
				default:
					logger.Infof("get event (key{%s}) = event{%d}", event.Kv.Key, event.Type)
				}
			}
		}
	}
}

func handleDeleteEvent(key, val []byte) {
	lock.Lock()
	defer lock.Unlock()

	keyStr := string(key)
	keyStr = strings.TrimSuffix(keyStr, "/")

	re := getCheckResourceRegexp()
	if m := re.Match(key); m {
		pathArray := strings.Split(keyStr, "/")
		if len(pathArray) == 0 {
			logger.Errorf("handleDeleteEvent key format error")
			return
		}
		resourceIdStr := pathArray[len(pathArray)-1]
		ID, err := strconv.Atoi(resourceIdStr)
		if err != nil {
			logger.Errorf("handleDeleteEvent ID is not int error %s", err)
			return
		}
		deleteApiConfigResource(ID)
		return
	}

	re = getExtractMethodRegexp()
	if m := re.Match(key); m {
		pathArray := strings.Split(keyStr, "/")
		if len(pathArray) < 3 {
			logger.Errorf("handleDeleteEvent key format error")
			return
		}
		resourceIdStr := pathArray[len(pathArray)-3]
		resourceId, err := strconv.Atoi(resourceIdStr)
		if err != nil {
			logger.Errorf("handleDeleteEvent ID is not int error %s", err)
			return
		}

		methodIdStr := pathArray[len(pathArray)-1]
		methodId, err := strconv.Atoi(methodIdStr)
		if err != nil {
			logger.Errorf("handleDeleteEvent ID is not int error %s", err)
			return
		}
		deleteApiConfigMethod(resourceId, methodId)
	}

	re = getCheckRatelimitRegexp()
	if m := re.Match(key); m {
		empty := &fr.Config{}
		listener.RateLimitChange(empty)
	}
}

func handlePutEvent(key, val []byte) {
	lock.Lock()
	defer lock.Unlock()

	re := getCheckResourceRegexp()
	if m := re.Match(key); m {
		res := &fc.Resource{}
		err := yaml.UnmarshalYML(val, res)
		if err != nil {
			logger.Errorf("handlePutEvent UnmarshalYML error %s", err)
			return
		}
		mergeApiConfigResource(*res)
		return
	}

	re = getExtractMethodRegexp()
	if m := re.Match(key); m {
		res := &fc.Method{}
		err := yaml.UnmarshalYML(val, res)
		if err != nil {
			logger.Errorf("handlePutEvent UnmarshalYML error %s", err)
			return
		}
		mergeApiConfigMethod(res.ResourcePath, *res)
		return
	}

	//handle base info
	re = getCheckBaseInfoRegexp()
	if m := re.Match(key); m {
		mergeBaseInfo(val)
		return
	}

	//handle ratelimit
	re = getCheckRatelimitRegexp()
	if m := re.Match(key); m {
		mergeRatelimit(val)
		return
	}
}

func deleteApiConfigResource(resourceId int) {
	for i := 0; i < len(apiConfig.Resources); i++ {
		itr := apiConfig.Resources[i]
		if itr.ID == resourceId {
			apiConfig.Resources = append(apiConfig.Resources[:i], apiConfig.Resources[i+1:]...)
			listener.ResourceDelete(itr)
			return
		}
	}
}

func mergeApiConfigResource(val fc.Resource) {
	for i, resource := range apiConfig.Resources {
		if val.ID != resource.ID {
			continue
		}
		// modify one resource
		val.Methods = resource.Methods
		apiConfig.Resources[i] = val
		listener.ResourceChange(val, resource)
		return
	}
	// add one resource
	apiConfig.Resources = append(apiConfig.Resources, val)
	listener.ResourceAdd(val)
}

func mergeRatelimit(val []byte) {
	c := &fr.Config{}
	if err := yaml.UnmarshalYML(val, c); err != nil {
		logger.Errorf("unmarshalYmlConfig error %s", err)
		return
	}
	apiConfig.RateLimit = *c
	listener.RateLimitChange(c)
}

func mergeBaseInfo(val []byte) {
	_ = initBaseInfoFromString(apiConfig, string(val))
}

func deleteApiConfigMethod(resourceId, methodId int) {
	for _, resource := range apiConfig.Resources {
		if resource.ID != resourceId {
			continue
		}

		for i := 0; i < len(resource.Methods); i++ {
			method := resource.Methods[i]

			if method.ID == methodId {
				resource.Methods = append(resource.Methods[:i], resource.Methods[i+1:]...)
				listener.MethodDelete(resource, method)
				return
			}
		}
	}
}

func mergeApiConfigMethod(path string, val fc.Method) {
	for i, resource := range apiConfig.Resources {
		if path != resource.Path {
			continue
		}

		for j, method := range resource.Methods {
			if method.ID == val.ID {
				// modify one method
				resource.Methods[j] = val
				listener.MethodChange(resource, val, method)
				apiConfig.Resources[i] = resource
				return
			}
		}
		// add one method
		resource.Methods = append(resource.Methods, val)
		apiConfig.Resources[i] = resource
		listener.MethodAdd(resource, val)
	}
}

func getCheckBaseInfoRegexp() *regexp.Regexp {
	return regexp.MustCompile(".+/base$")
}

func getCheckResourceRegexp() *regexp.Regexp {
	return regexp.MustCompile(".+/resources/[^/]+/?$")
}

func getExtractMethodRegexp() *regexp.Regexp {
	return regexp.MustCompile(".+/resources/([^/]+)/method/[^/]+/?$")
}

func getCheckRatelimitRegexp() *regexp.Regexp {
	return regexp.MustCompile(".+/filter/ratelimit")
}

// RegisterConfigListener register APIConfigListener
func RegisterConfigListener(li APIConfigResourceListener) {
	listener = li
}

// GetAPIConf returns the init api config
func GetAPIConf() fc.APIConfig {
	return *apiConfig
}
