#
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.
#

cur_mkfile := $(abspath $(lastword $(MAKEFILE_LIST)))
currentPath := $(patsubst %/, %, $(dir $(cur_mkfile)))
pixiuPath := /cmd/pixiu/
mainPath := $(currentPath)$(pixiuPath)

targetName := dubbo-go-pixiu

api-config-path:=${api-config}
ifeq (,$(api-config-path))
$(warning api-config-path is nil, default: configs/api_config.yaml)
api-config-path = configs/api_config.yaml
endif

config-path:=${config-path}
ifeq (,$(config-path))
$(warning config-path is nil, default: configs/conf.yaml)
config-path = configs/conf.yaml
endif

$(info api-config-path = $(api-config-path))
$(info config-path = $(config-path))

os := $(shell go env GOOS)
ifeq (windows,$(os))
	targetName = dubbo-go-pixiu.exe
endif
exe := $(mainPath)$(targetName)
build:
	cd $(mainPath) && go build  -o $(targetName)

run: build
	cp $(exe) $(currentPath) && ./dubbo-go-pixiu start -a $(api-config-path) -c $(config-path)

license-check-util:
	go install github.com/lsm-dev/license-header-checker/cmd/license-header-checker@latest

license-check:
	license-header-checker -v -a -r -i vendor -i .github/actions /tmp/tools/license/license.txt . go

test:
	sh before_ut.sh
	go test ./pkg/... -coverprofile=coverage.txt -covermode=atomic

integrate-test:
	sh start_integrate_test.sh