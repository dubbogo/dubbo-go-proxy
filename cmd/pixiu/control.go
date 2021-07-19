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

package main

import (
	"runtime"
)

import (
	"github.com/urfave/cli"
)

import (
	"github.com/apache/dubbo-go-pixiu/pkg/config"
	"github.com/apache/dubbo-go-pixiu/pkg/logger"
	"github.com/apache/dubbo-go-pixiu/pkg/pixiu"
)

var (
	flagToLogLevel = map[string]string{
		"trace":    "TRACE",
		"debug":    "DEBUG",
		"info":     "INFO",
		"warning":  "WARN",
		"error":    "ERROR",
		"critical": "FATAL",
	}

	cmdStart = cli.Command{
		Name:  "start",
		Usage: "start dubbogo pixiu",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "config, c",
				Usage:  "Load configuration from `FILE`",
				EnvVar: "DUBBOGO_PIXIU_CONFIG",
				Value:  "configs/conf.yaml",
			},
			cli.StringFlag{
				Name:   "api-config, a",
				Usage:  "Load api configuration from `FILE`",
				EnvVar: "DUBBOGO_PIXIU_API_CONFIG",
				Value:  "configs/api_config.yaml",
			},
			cli.StringFlag{
				Name:   "log-config, lc",
				Usage:  "Load log configuration from `FILE`",
				EnvVar: "LOG_FILE",
				Value:  "configs/log.yml",
			},
			cli.StringFlag{
				Name:   "log-level, l",
				Usage:  "dubbogo pixiu log level, trace|debug|info|warning|error|critical",
				EnvVar: "LOG_LEVEL",
			},
			cli.StringFlag{
				Name:  "log-format, lf",
				Usage: "dubbogo pixiu log format, currently useless",
			},
			cli.StringFlag{
				Name:  "limit-cpus, limc",
				Usage: "dubbogo pixiu schedule threads count",
			},
		},
		Action: func(c *cli.Context) error {
			configPath := c.String("config")
			apiConfigPath := c.String("api-config")
			flagLogLevel := c.String("log-level")
			logConfPath := c.String("log-config")

			err := logger.InitLog(logConfPath)
			if err != nil {
				return err
			}

			bootstrap := config.Load(configPath)
			if logLevel, ok := flagToLogLevel[flagLogLevel]; ok {
				logger.SetLoggerLevel(logLevel)
			}

			initFromRemote := false
			if bootstrap.GetAPIMetaConfig() != nil {
				if _, err := config.LoadAPIConfig(bootstrap.GetAPIMetaConfig()); err != nil {
					logger.Warnf("load api config from etcd error:%+v", err)
				} else {
					initFromRemote = true
				}
			}

			if !initFromRemote {
				if _, err := config.LoadAPIConfigFromFile(apiConfigPath); err != nil {
					logger.Errorf("load api config error:%+v", err)
					return err
				}
			}

			limitCpus := c.Int("limit-cpus")
			if limitCpus <= 0 {
				runtime.GOMAXPROCS(runtime.NumCPU())
			} else {
				runtime.GOMAXPROCS(limitCpus)
			}

			pixiu.Start(bootstrap)
			return nil
		},
	}

	cmdStop = cli.Command{
		Name:  "stop",
		Usage: "stop dubbogo pixiu",
		Action: func(c *cli.Context) error {
			return nil
		},
	}

	cmdReload = cli.Command{
		Name:  "reload",
		Usage: "reconfiguration",
		Action: func(c *cli.Context) error {
			return nil
		},
	}
)
