#!/bin/sh
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
if [ "$1" != "skip_zookeeper" ]; then
  zookeeper_container_id=$(docker ps | grep zookeeper | head -n 1 | awk '{print  $1}')
  if [ -n "$zookeeper_container_id" ]; then
    echo "removing old zookeeper!"
    docker kill "$zookeeper_container_id"
    docker rm "$zookeeper_container_id"
  fi
  echo "starting zookeeper!"
  docker run -dit --name zookeeper -p 2181:2181 zookeeper:3.4.14
  echo "zookeeper stated!"
fi

go env -w GOPROXY=https://goproxy.cn,direct

echo "starting dubbogo provider!"
if [ -z "$CONF_PROVIDER_FILE_PATH" ]; then
  export CONF_PROVIDER_FILE_PATH=../../sample/http/server/config/server.yml
fi
go build ../../sample/http/server/app/

provider_pid=$(ps -ef | grep ../../sample/http/server/app/ | grep -v 'grep' | awk '{print $2}')

if [ -n "$provider_pid" ]; then
  echo "pid of old dubbogo provider is $provider_pid, kill it"
  kill -9 "$provider_pid"
fi
nohup go run ../../sample/http/server/app/ >http_server.out &
sleep 10
echo "dubbogo provider started!"

## to start pixiu

echo "starting proxy!"

cd ../../
make run config-path=sample/http/proxy/conf.yaml api-config-path=sample/http/proxy/api_config.yaml
