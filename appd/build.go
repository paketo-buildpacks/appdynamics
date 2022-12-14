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

	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
)

type Build struct {
	Logger bard.Logger
}

func (b Build) Build(context libcnb.BuildContext) (libcnb.BuildResult, error) {
	b.Logger.Title(context.Buildpack)
	result := libcnb.NewBuildResult()

	cr, err := libpak.NewConfigurationResolver(context.Buildpack, &b.Logger)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create configuration resolver\n%w", err)
	}

	pr := libpak.PlanEntryResolver{Plan: context.Plan}

	dr, err := libpak.NewDependencyResolver(context)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency resolver\n%w", err)
	}

	dc, err := libpak.NewDependencyCache(context)
	if err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to create dependency cache\n%w", err)
	}
	dc.Logger = b.Logger

	if _, ok, err := pr.Resolve("appdynamics-java"); err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to resolve appdynamics-java plan entry\n%w", err)
	} else if ok {
		agentDependency, err := dr.Resolve("appdynamics-java", "")
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to find dependency\n%w", err)
		}

		var externalConfigurationDependency *libpak.BuildpackDependency
		if uri, ok := cr.Resolve("BP_APPD_EXT_CONF_URI"); ok {
			v, _ := cr.Resolve("BP_APPD_EXT_CONF_VERSION")
			s, _ := cr.Resolve("BP_APPD_EXT_CONF_SHA256")

			externalConfigurationDependency = &libpak.BuildpackDependency{
				ID:      "appdynamics-external-configuration",
				Name:    "AppDynamics External Configuration",
				Version: v,
				URI:     uri,
				SHA256:  s,
				Stacks:  []string{context.StackID},
				CPEs:    []string{fmt.Sprintf("cpe:2.3:a:appdynamics:external-configuration:%s:*:*:*:*:*:*:*", v)},
				PURL:    fmt.Sprintf("pkg:generic/appdynamics-external-configuration@%s", v),
			}
		}

		ja, bes := NewJavaAgent(context.Buildpack.Path, agentDependency, cr, externalConfigurationDependency, dc)
		ja.Logger = b.Logger
		result.Layers = append(result.Layers, ja)
		for _, be := range bes {
			result.BOM.Entries = append(result.BOM.Entries, be)
		}
	}

	if _, ok, err := pr.Resolve("appdynamics-nodejs"); err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to resolve appdynamics-nodejs plan entry\n%w", err)
	} else if ok {
		dep, err := dr.Resolve("appdynamics-nodejs", "")
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to find dependency\n%w", err)
		}

		na, be := NewNodeJSAgent(context.Application.Path, context.Buildpack.Path, dep, dc)
		na.Logger = b.Logger
		result.Layers = append(result.Layers, na)
		result.BOM.Entries = append(result.BOM.Entries, be)
	}

	if _, ok, err := pr.Resolve("appdynamics-php"); err != nil {
		return libcnb.BuildResult{}, fmt.Errorf("unable to resolve appdynamics-php plan entry\n%w", err)
	} else if ok {
		dep, err := dr.Resolve("appdynamics-php", "")
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to find dependency\n%w", err)
		}

		pa, be := NewPHPAgent(dep, dc)
		pa.Logger = b.Logger
		result.Layers = append(result.Layers, pa)
		result.BOM.Entries = append(result.BOM.Entries, be)
	}

	h, be := libpak.NewHelperLayer(context.Buildpack, "properties")
	h.Logger = b.Logger
	result.Layers = append(result.Layers, h)
	result.BOM.Entries = append(result.BOM.Entries, be)

	return result, nil
}
