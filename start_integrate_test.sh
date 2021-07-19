#
#  Licensed to the Apache Software Foundation (ASF) under one or more
#  contributor license agreements.  See the NOTICE file distributed with
#  this work for additional information regarding copyright ownership.
#  The ASF licenses this file to You under the Apache License, Version 2.0
#  (the "License"); you may not use this file except in compliance with
#  the License.  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

array=("samples/dubbogo/simple/body")
array+=("samples/dubbogo/simple/mix")
array+=("samples/dubbogo/simple/proxy")
array+=("samples/dubbogo/simple/query")
array+=("samples/dubbogo/simple/uri")

#http



for((i=0;i<${#array[*]};i++))
do
	sh ./integrate_test.sh ${array[i]}
	result=$?
	if [ $result -gt 0 ]; then
    exit $result
	fi
done
