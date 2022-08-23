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
	"strconv"

	"github.com/paketo-buildpacks/libpak/sbom"

	"github.com/buildpacks/libcnb"
	"github.com/heroku/color"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/crush"
	"github.com/paketo-buildpacks/libpak/sherpa"
)

type JavaAgent struct {
	AgentDependency                 libpak.BuildpackDependency
	BuildpackPath                   string
	ConfigurationResolver           libpak.ConfigurationResolver
	DependencyCache                 libpak.DependencyCache
	ExternalConfigurationDependency *libpak.BuildpackDependency
	LayerContributor                libpak.LayerContributor
	Logger                          bard.Logger
}

func NewJavaAgent(buildpackPath string, agentDependency libpak.BuildpackDependency, configurationResolver libpak.ConfigurationResolver, externalConfigurationDependency *libpak.BuildpackDependency, cache libpak.DependencyCache) (JavaAgent, []libcnb.BOMEntry) {

	dependencies := []libpak.BuildpackDependency{agentDependency}

	if externalConfigurationDependency != nil {
		dependencies = append(dependencies, *externalConfigurationDependency)
	}

	j := JavaAgent{
		AgentDependency:                 agentDependency,
		BuildpackPath:                   buildpackPath,
		ConfigurationResolver:           configurationResolver,
		DependencyCache:                 cache,
		ExternalConfigurationDependency: externalConfigurationDependency,
		LayerContributor: libpak.NewLayerContributor(
			fmt.Sprintf("%s %s", agentDependency.Name, agentDependency.Version),
			map[string]interface{}{
				"dependencies": dependencies,
			},
			libcnb.LayerTypes{Launch: true},
		),
	}

	var bomEntries []libcnb.BOMEntry
	entry := agentDependency.AsBOMEntry()
	entry.Metadata["layer"] = j.Name()
	entry.Launch = true
	bomEntries = append(bomEntries, entry)

	if externalConfigurationDependency != nil {
		entry := externalConfigurationDependency.AsBOMEntry()
		entry.Metadata["layer"] = j.Name()
		entry.Launch = true
		bomEntries = append(bomEntries, entry)
	}

	return j, bomEntries
}

func (j JavaAgent) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	j.LayerContributor.Logger = j.Logger
	var syftArtifacts []sbom.SyftArtifact

	return j.LayerContributor.Contribute(layer, func() (libcnb.Layer, error) {
		if err := j.ContributeAgent(layer); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to contribute agent\n%w", err)
		}
		if syftArtifact, err := j.AgentDependency.AsSyftArtifact(); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to get Syft Artifact for dependency: %s, \n%w", j.AgentDependency.Name, err)
		} else {
			syftArtifacts = append(syftArtifacts, syftArtifact)
		}

		if err := j.ContributeConfiguration(layer); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to contribute configuration\n%w", err)
		}

		if j.ExternalConfigurationDependency != nil {
			if err := j.ContributeExternalConfiguration(layer); err != nil {
				return libcnb.Layer{}, fmt.Errorf("unable to contribute external configuration\n%w", err)
			}
			if syftArtifact, err := j.ExternalConfigurationDependency.AsSyftArtifact(); err != nil {
				return libcnb.Layer{}, fmt.Errorf("unable to get Syft Artifact for dependency: %s, \n%w", j.ExternalConfigurationDependency.Name, err)
			} else {
				syftArtifacts = append(syftArtifacts, syftArtifact)
			}
		}

		layer.LaunchEnvironment.Appendf("JAVA_TOOL_OPTIONS", " ",
			"-javaagent:%s", filepath.Join(layer.Path, "javaagent.jar"))

		if err := j.writeDependencySBOM(layer, syftArtifacts); err != nil {
			return libcnb.Layer{}, err
		}

		return layer, nil
	})
}

func (j JavaAgent) ContributeAgent(layer libcnb.Layer) error {
	artifact, err := j.DependencyCache.Artifact(j.AgentDependency)
	if err != nil {
		return fmt.Errorf("unable to get dependency %s\n%w", j.AgentDependency.ID, err)
	}
	defer artifact.Close()

	j.Logger.Bodyf("Expanding to %s", layer.Path)

	if err := crush.ExtractZip(artifact, layer.Path, 0); err != nil {
		return fmt.Errorf("unable to extract to %s\n%w", layer.Path, err)
	}

	return nil
}

func (j JavaAgent) ContributeConfiguration(layer libcnb.Layer) error {
	v, err := VersionDirectory(layer)
	if err != nil {
		return fmt.Errorf("unable to determine version directory\n%w", err)
	}

	file := filepath.Join(v, "conf")
	if err := os.MkdirAll(file, 0755); err != nil {
		return fmt.Errorf("unable to create directory %s\n%w", file, err)
	}

	j.Logger.Bodyf("Copying app-agent-config.xml to %s/conf", v)
	file = filepath.Join(j.BuildpackPath, "resources", "app-agent-config.xml")
	in, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("unable to open %s\n%w", file, err)
	}
	defer in.Close()

	file = filepath.Join(v, "conf", "app-agent-config.xml")
	if err := sherpa.CopyFile(in, file); err != nil {
		return fmt.Errorf("unable to copy %s to %s\n%w", in.Name(), file, err)
	}

	j.Logger.Bodyf("Copying custom-activity-correlation.xml to %s/conf", v)
	file = filepath.Join(j.BuildpackPath, "resources", "custom-activity-correlation.xml")
	in, err = os.Open(file)
	if err != nil {
		return fmt.Errorf("unable to open %s\n%w", file, err)
	}
	defer in.Close()

	file = filepath.Join(v, "conf", "custom-activity-correlation.xml")
	if err := sherpa.CopyFile(in, file); err != nil {
		return fmt.Errorf("unable to copy %s to %s\n%w", in.Name(), file, err)
	}

	file = filepath.Join(v, "conf", "logging")
	if err := os.MkdirAll(file, 0755); err != nil {
		return fmt.Errorf("unable to create directory %s\n%w", file, err)
	}

	j.Logger.Bodyf("Copying log4j2.xml to %s/conf/logging", v)
	file = filepath.Join(j.BuildpackPath, "resources", "log4j2.xml")
	in, err = os.Open(file)
	if err != nil {
		return fmt.Errorf("unable to open %s\n%w", file, err)
	}
	defer in.Close()

	file = filepath.Join(v, "conf", "logging", "log4j2.xml")
	if err := sherpa.CopyFile(in, file); err != nil {
		return fmt.Errorf("unable to copy %s to %s\n%w", in.Name(), file, err)
	}

	logDir := filepath.Join(v, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("unable to create log directory %s\n%w", logDir, err)
	}
	// Make version*/logs directory group-writable so that a custom user can run container image (must still be cnb group)
	if err := os.Chmod(logDir, 0775); err != nil {
		return fmt.Errorf("unable to set log directory permissions %s\n%w", logDir, err)
	}

	return nil
}

func (j JavaAgent) ContributeExternalConfiguration(layer libcnb.Layer) error {
	j.Logger.Header(color.BlueString("%s %s", j.ExternalConfigurationDependency.Name, j.ExternalConfigurationDependency.Version))

	artifact, err := j.DependencyCache.Artifact(*j.ExternalConfigurationDependency)
	if err != nil {
		return fmt.Errorf("unable to get dependency %s\n%w", j.ExternalConfigurationDependency.ID, err)
	}
	defer artifact.Close()

	j.Logger.Bodyf("Expanding to %s", layer.Path)

	c := 0
	if s, ok := j.ConfigurationResolver.Resolve("BP_APPD_EXT_CONF_STRIP"); ok {
		if c, err = strconv.Atoi(s); err != nil {
			return fmt.Errorf("unable to parse %s to integer\n%w", s, err)
		}
	}

	v, err := VersionDirectory(layer)
	if err != nil {
		return fmt.Errorf("unable to determine version directory\n%w", err)
	}

	if err := crush.ExtractTarGz(artifact, v, c); err != nil {
		return fmt.Errorf("unable to expand external configuration\n%w", err)
	}

	return nil
}

func (j JavaAgent) writeDependencySBOM(layer libcnb.Layer, syftArtifacts []sbom.SyftArtifact) error {

	sbomPath := layer.SBOMPath(libcnb.SyftJSON)
	dep := sbom.NewSyftDependency(layer.Path, syftArtifacts)
	j.Logger.Debugf("Writing Syft SBOM at %s: %+v", sbomPath, dep)
	if err := dep.WriteTo(sbomPath); err != nil {
		return fmt.Errorf("unable to write SBOM\n%w", err)
	}
	return nil
}

func (JavaAgent) Name() string {
	return "appdynamics-java"
}
