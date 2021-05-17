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

package matcher

import (
	"github.com/apache/dubbo-go-pixiu/pkg/filter/ratelimit/config"
)

import (
	"regexp"
	"sync"
)

type Regex struct {
	apiNames map[string]string
	sync.RWMutex
}

func (p *Regex) load(apis []config.APIResource) {
	m := map[string]string{}

	for _, api := range apis {
		apiName := api.Name
		for _, api := range api.Items {
			if api.MatchStrategy == config.REGEX {
				m[api.Pattern] = apiName
			}
		}
	}

	p.Lock()
	defer p.Unlock()
	p.apiNames = m
}

func (p *Regex) match(path string) (string, bool) {
	p.RLock()
	defer p.RUnlock()

	for k, v := range p.apiNames {
		matched, _ := regexp.MatchString(k, path)
		if matched {
			return v, true
		}
	}
	return "", false
}