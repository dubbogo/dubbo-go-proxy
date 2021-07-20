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

if [ -z "$1" ]; then
  echo 'Provide test directory please, like : ./integrate_test.sh $(pwd)/samples/simple/server .'
  exit
fi

P_DIR=$(pwd)/$1
PIXIU_DIR=$(pwd)

make PROJECT_DIR=$P_DIR PIXIU_DIR=$PIXIU_DIR PROJECT_NAME=$(basename $P_DIR) BASE_DIR=$P_DIR/dist -f igt/Makefile docker-up

# start server
make PROJECT_DIR=$P_DIR PIXIU_DIR=$PIXIU_DIR PROJECT_NAME=$(basename $P_DIR) BASE_DIR=$P_DIR/dist -f igt/Makefile start
sleep 0.5
# start pixiu
make PROJECT_DIR=$P_DIR PIXIU_DIR=$PIXIU_DIR PROJECT_NAME=$(basename $P_DIR) BASE_DIR=$P_DIR/dist -f igt/Makefile buildPixiu
sleep 0.5
# start integration
make PROJECT_DIR=$P_DIR PIXIU_DIR=$PIXIU_DIR PROJECT_NAME=$(basename $P_DIR) BASE_DIR=$P_DIR/dist -f igt/Makefile integration
result=$?
# stop server
make PROJECT_DIR=$P_DIR PIXIU_DIR=$PIXIU_DIR PROJECT_NAME=$(basename $P_DIR) BASE_DIR=$P_DIR/dist -f igt/Makefile clean

make PROJECT_DIR=$P_DIR PIXIU_DIR=$PIXIU_DIR PROJECT_NAME=$(basename $P_DIR) BASE_DIR=$P_DIR/dist -f igt/Makefile docker-down

exit $((result))
