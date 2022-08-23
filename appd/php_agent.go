/*
 * Copyright 2018-2022 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package appd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/crush"
	"github.com/paketo-buildpacks/libpak/effect"
)

type PHPAgent struct {
	Executor         effect.Executor
	LayerContributor libpak.DependencyLayerContributor
	Logger           bard.Logger
}

func NewPHPAgent(dependency libpak.BuildpackDependency, cache libpak.DependencyCache) (PHPAgent, libcnb.BOMEntry) {
	contributor, entry := libpak.NewDependencyLayer(dependency, cache, libcnb.LayerTypes{Launch: true})
	return PHPAgent{
		Executor:         effect.NewExecutor(),
		LayerContributor: contributor,
	}, entry
}

func (p PHPAgent) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	p.LayerContributor.Logger = p.Logger

	return p.LayerContributor.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
		p.Logger.Bodyf("Expanding to %s", layer.Path)

		if err := crush.ExtractTarBz2(artifact, layer.Path, 1); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to expand New Relic\n%w", err)
		}

		file := filepath.Join(layer.Path, "php.ini.d")
		if err := os.MkdirAll(file, 0755); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to create %s\n%w", file, err)
		}

		layer.LaunchEnvironment.Prepend("PHP_INI_SCAN_DIR", string(os.PathListSeparator), file)

		e, ok := os.LookupEnv("PHP_EXTENSION_DIR")
		if !ok {
			return libcnb.Layer{}, fmt.Errorf("unable to find $PHP_EXTENSION_DIR")
		}

		if err := p.Executor.Execute(effect.Execution{
			Command: filepath.Join(layer.Path, "install.sh"),
			Args: []string{
				"--ignore-permissions",
				fmt.Sprintf("--php-extension-dir=%s", e),
				fmt.Sprintf("--php-ini-dir=%s", file),
				"--account-info=${APPDYNAMICS_AGENT_ACCOUNT_NAME}@${APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY}",
				"${APPDYNAMICS_CONTROLLER_HOST_NAME}",
				"${APPDYNAMICS_CONTROLLER_PORT}",
				"${APPDYNAMICS_AGENT_APPLICATION_NAME}",
				"${APPDYNAMICS_AGENT_TIER_NAME}",
				"${APPDYNAMICS_AGENT_NODE_NAME}",
			},
			Dir:    layer.Path,
			Stdout: p.Logger.InfoWriter(),
			Stderr: p.Logger.InfoWriter(),
		}); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to run install.sh\n%w", err)
		}

		file = filepath.Join(layer.Path, "php.ini.d")
		if err := os.MkdirAll(file, 0755); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to create %s\n%w", file, err)
		}

		file = filepath.Join(layer.Path, "php.ini.d", "appdynamics_agent.ini")
		out, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to open %s\n%w", file, err)
		}
		defer out.Close()

		_, err = out.WriteString("\nagent.controller.ssl.enabled = ${APPDYNAMICS_CONTROLLER_SSL_ENABLED}\n")
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to append file %s\n%w", file, err)
		}

		return layer, nil
	})
}

func (p PHPAgent) Name() string {
	return p.LayerContributor.LayerName()
}
