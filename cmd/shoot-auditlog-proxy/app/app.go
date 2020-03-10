// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"fmt"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/logger"
	proxyconf "github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/proxy/config"
	"github.com/gardener/gardener-extension-shoot-auditlog-service/pkg/proxy/webhook"
	"os"

	"github.com/spf13/cobra"
)

// NewServiceControllerCommand creates a new command that is used to start the Certificate Service controller.
func NewProxyServerCommand() *cobra.Command {
	proxyOptions := proxyconf.AuditlogProxyOptions{}

	cmd := &cobra.Command{
		Use:   "auditlog-proxy",
		Short: "Auditlog Proxy receives auditlog events from a kube apiserver and writes them to a storage backend.",

		Run: func(cmd *cobra.Command, args []string) {

			log, err := logger.New(nil)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			if err := proxyOptions.Complete(); err != nil {
				log.Error(err, "unable to parse configuration")
				os.Exit(1)
			}
			if err := webhook.Run(log, proxyOptions.Completed()); err != nil {
				log.Error(err, "unable to start webhook server")
				os.Exit(1)
			}
		},
	}

	logger.AddFlags(cmd.Flags())
	proxyOptions.AddFlags(cmd.Flags())

	return cmd
}
