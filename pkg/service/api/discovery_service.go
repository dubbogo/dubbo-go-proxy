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

package api

import (
	"errors"
	"fmt"
	"strings"
)

import (
	"github.com/apache/dubbo-go-pixiu/pkg/common/constant"
	"github.com/apache/dubbo-go-pixiu/pkg/common/extension"
	pc "github.com/apache/dubbo-go-pixiu/pkg/config"
	"github.com/apache/dubbo-go-pixiu/pkg/filter/ratelimit"
	"github.com/apache/dubbo-go-pixiu/pkg/router"
	"github.com/apache/dubbo-go-pixiu/pkg/service"
)

import (
	"github.com/dubbogo/dubbo-go-pixiu-filter/pkg/api/config"
	ratelimitConf "github.com/dubbogo/dubbo-go-pixiu-filter/pkg/api/config/ratelimit"
	fr "github.com/dubbogo/dubbo-go-pixiu-filter/pkg/router"
)

// Init set api discovery local_memory service.
func Init() {
	extension.SetAPIDiscoveryService(constant.LocalMemoryApiDiscoveryService, NewLocalMemoryAPIDiscoveryService())
}

// LocalMemoryAPIDiscoveryService is the local cached API discovery service
type LocalMemoryAPIDiscoveryService struct {
	router *router.Route
}

// NewLocalMemoryAPIDiscoveryService creates a new LocalMemoryApiDiscoveryService instance
func NewLocalMemoryAPIDiscoveryService() *LocalMemoryAPIDiscoveryService {
	return &LocalMemoryAPIDiscoveryService{
		router: router.NewRoute(),
	}
}

// AddAPI adds a method to the router tree
func (l *LocalMemoryAPIDiscoveryService) AddAPI(api fr.API) error {
	return l.router.PutAPI(api)
}

// GetAPI returns the method to the caller
func (l *LocalMemoryAPIDiscoveryService) GetAPI(url string, httpVerb config.HTTPVerb) (fr.API, error) {
	if api, ok := l.router.FindAPI(url, httpVerb); ok {
		return *api, nil
	}

	return fr.API{}, errors.New("not found")
}

// ClearAPI clear all api
func (l *LocalMemoryAPIDiscoveryService) ClearAPI() error {
	return l.router.ClearAPI()
}

// RemoveAPIByPath remove all api belonged to path
func (l *LocalMemoryAPIDiscoveryService) RemoveAPIByPath(deleted config.Resource) error {
	_, groupPath := getDefaultPath()
	fullPath := getFullPath(groupPath, deleted.Path)

	l.router.DeleteNode(fullPath)
	return nil
}

// RemoveAPIByPath remove all api
func (l *LocalMemoryAPIDiscoveryService) RemoveAPI(fullPath string, method config.Method) error {
	l.router.DeleteAPI(fullPath, method.HTTPVerb)
	return nil
}

// ResourceChange handle modify resource event
func (l *LocalMemoryAPIDiscoveryService) ResourceChange(new config.Resource, old config.Resource) bool {
	if err := modifyAPIFromResource(new, old, l); err == nil {
		return true
	}
	return false
}

// ResourceAdd handle add resource event
func (l *LocalMemoryAPIDiscoveryService) ResourceAdd(res config.Resource) bool {
	parentPath, groupPath := getDefaultPath()

	fullHeaders := make(map[string]string, 9)
	if err := addAPIFromResource(res, l, groupPath, parentPath, fullHeaders); err == nil {
		return true
	}
	return false
}

// ResourceDelete handle delete resource event
func (l *LocalMemoryAPIDiscoveryService) ResourceDelete(deleted config.Resource) bool {
	if err := deleteAPIFromResource(deleted, l); err == nil {
		return true
	}
	return false
}

// MethodChange handle modify method event
func (l *LocalMemoryAPIDiscoveryService) MethodChange(res config.Resource, new config.Method, old config.Method) bool {
	_, groupPath := getDefaultPath()
	fullPath := getFullPath(groupPath, res.Path)
	fullHeaders := make(map[string]string, 9)
	if err := modifyAPIFromMethod(fullPath, new, old, fullHeaders, l); err == nil {
		return true
	}
	return false
}

// MethodAdd handle add method event
func (l *LocalMemoryAPIDiscoveryService) MethodAdd(res config.Resource, method config.Method) bool {
	_, groupPath := getDefaultPath()
	fullPath := getFullPath(groupPath, res.Path)
	fullHeaders := make(map[string]string, 9)
	if err := addAPIFromMethod(fullPath, method, fullHeaders, l); err == nil {
		return true
	}
	return false
}

// MethodDelete handle delete method event
func (l *LocalMemoryAPIDiscoveryService) MethodDelete(res config.Resource, method config.Method) bool {
	_, groupPath := getDefaultPath()
	fullPath := getFullPath(groupPath, res.Path)
	if err := deleteAPIFromMethod(fullPath, method, l); err == nil {
		return true
	}
	return false
}

// InitAPIsFromConfig inits the router from API config and to local cache
func InitAPIsFromConfig(apiConfig config.APIConfig) error {
	localAPIDiscSrv := extension.GetMustAPIDiscoveryService(constant.LocalMemoryApiDiscoveryService)
	if len(apiConfig.Resources) == 0 {
		return nil
	}
	// register config change listener
	pc.RegisterConfigListener(localAPIDiscSrv)
	return loadAPIFromResource("", apiConfig.Resources, nil, localAPIDiscSrv)
}

func loadAPIFromResource(parentPath string, resources []config.Resource, parentHeaders map[string]string, localSrv service.APIDiscoveryService) error {
	errStack := []string{}
	if len(resources) == 0 {
		return nil
	}
	groupPath := parentPath
	if parentPath == constant.PathSlash {
		groupPath = ""
	}
	fullHeaders := parentHeaders
	if fullHeaders == nil {
		fullHeaders = make(map[string]string, 9)
	}
	for _, resource := range resources {
		err := addAPIFromResource(resource, localSrv, groupPath, parentPath, fullHeaders)
		if err != nil {
			errStack = append(errStack, err.Error())
		}
	}
	if len(errStack) > 0 {
		return errors.New(strings.Join(errStack, "; "))
	}
	return nil
}

func getDefaultPath() (string, string) {
	return "", ""
}

func modifyAPIFromResource(new config.Resource, old config.Resource, localSrv service.APIDiscoveryService) error {
	parentPath, groupPath := getDefaultPath()
	fullHeaders := make(map[string]string, 9)

	err := deleteAPIFromResource(old, localSrv)
	if err != nil {
		return err
	}

	err = addAPIFromResource(new, localSrv, groupPath, parentPath, fullHeaders)
	return err
}

func deleteAPIFromResource(old config.Resource, localSrv service.APIDiscoveryService) error {
	return localSrv.RemoveAPIByPath(old)
}

func addAPIFromResource(resource config.Resource, localSrv service.APIDiscoveryService, groupPath string, parentPath string, fullHeaders map[string]string) error {
	fullPath := getFullPath(groupPath, resource.Path)
	if !strings.HasPrefix(resource.Path, constant.PathSlash) {
		return fmt.Errorf("path %s in %s doesn't start with /", resource.Path, parentPath)
	}
	for headerName, headerValue := range resource.Headers {
		fullHeaders[headerName] = headerValue
	}
	if len(resource.Resources) > 0 {
		if err := loadAPIFromResource(resource.Path, resource.Resources, fullHeaders, localSrv); err != nil {
			return err
		}
	}

	if err := loadAPIFromMethods(fullPath, resource.Methods, fullHeaders, localSrv); err != nil {
		return err
	}
	return nil
}

func addAPIFromMethod(fullPath string, method config.Method, headers map[string]string, localSrv service.APIDiscoveryService) error {
	api := fr.API{
		URLPattern: fullPath,
		Method:     method,
		Headers:    headers,
	}
	if err := localSrv.AddAPI(api); err != nil {
		return fmt.Errorf("path: %s, Method: %s, error: %s", fullPath, method.HTTPVerb, err.Error())
	}
	return nil
}

func modifyAPIFromMethod(fullPath string, new config.Method, old config.Method, headers map[string]string, localSrv service.APIDiscoveryService) error {
	if err := localSrv.RemoveAPI(fullPath, old); err != nil {
		return err
	}

	if err := addAPIFromMethod(fullPath, new, headers, localSrv); err != nil {
		return err
	}

	return nil
}

func deleteAPIFromMethod(fullPath string, deleted config.Method, localSrv service.APIDiscoveryService) error {
	return localSrv.RemoveAPI(fullPath, deleted)
}

func getFullPath(groupPath string, resourcePath string) string {
	return groupPath + resourcePath
}

func loadAPIFromMethods(fullPath string, methods []config.Method, headers map[string]string, localSrv service.APIDiscoveryService) error {
	errStack := []string{}
	for _, method := range methods {

		if err := addAPIFromMethod(fullPath, method, headers, localSrv); err != nil {
			errStack = append(errStack, err.Error())
		}
	}
	if len(errStack) > 0 {
		return errors.New(strings.Join(errStack, "\n"))
	}
	return nil
}

func (l *LocalMemoryAPIDiscoveryService) RateLimitChange(c *ratelimitConf.Config) {
	ratelimit.OnUpdate(c)
}
