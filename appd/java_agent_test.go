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

package appd_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/appdynamics/v4/appd"
)

func testJavaAgent(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx libcnb.BuildContext
	)

	it.Before(func() {
		var err error

		ctx.Buildpack.Path, err = ioutil.TempDir("", "java-agent-buildpack")
		Expect(err).NotTo(HaveOccurred())

		ctx.Layers.Path, err = ioutil.TempDir("", "java-agent-layers")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Buildpack.Path)).To(Succeed())
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
	})

	it("contributes Java agent", func() {
		Expect(os.MkdirAll(filepath.Join(ctx.Buildpack.Path, "resources"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(ctx.Buildpack.Path, "resources", "app-agent-config.xml"), []byte{}, 0644)).
			To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(ctx.Buildpack.Path, "resources", "custom-activity-correlation.xml"), []byte{}, 0644)).
			To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(ctx.Buildpack.Path, "resources", "log4j2.xml"), []byte{}, 0644)).
			To(Succeed())

		dep := libpak.BuildpackDependency{
			ID:     "appdynamics-java",
			URI:    "https://localhost/stub-appdynamics-agent.zip",
			SHA256: "ee23306ce5f7086219c1876652ed323970ebc249f21d1c79b737ac1120284bbf",
		}
		dc := libpak.DependencyCache{CachePath: "testdata"}

		j, bomEntries := appd.NewJavaAgent(ctx.Buildpack.Path, dep, libpak.ConfigurationResolver{}, nil, dc)
		Expect(bomEntries).To(HaveLen(1))
		Expect(bomEntries[0].Name).To(Equal("appdynamics-java"))
		Expect(bomEntries[0].Metadata["layer"]).To(Equal("appdynamics-java"))
		Expect(bomEntries[0].Launch).To(BeTrue())
		Expect(bomEntries[0].Build).To(BeFalse())

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = j.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(layer.Launch).To(BeTrue())
		Expect(filepath.Join(layer.Path, "javaagent.jar")).To(BeARegularFile())
		Expect(filepath.Join(layer.Path, "ver4.5.7.25056", "conf", "app-agent-config.xml")).To(BeARegularFile())
		Expect(filepath.Join(layer.Path, "ver4.5.7.25056", "conf", "custom-activity-correlation.xml")).To(BeARegularFile())
		Expect(filepath.Join(layer.Path, "ver4.5.7.25056", "conf", "logging", "log4j2.xml")).To(BeARegularFile())
		Expect(layer.LaunchEnvironment["JAVA_TOOL_OPTIONS.delim"]).To(Equal(" "))
		Expect(layer.LaunchEnvironment["JAVA_TOOL_OPTIONS.append"]).To(Equal(fmt.Sprintf("-javaagent:%s",
			filepath.Join(layer.Path, "javaagent.jar"))))
	})

	it("contributes external configuration", func() {
		Expect(os.MkdirAll(filepath.Join(ctx.Buildpack.Path, "resources"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(ctx.Buildpack.Path, "resources", "app-agent-config.xml"), []byte{}, 0644)).
			To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(ctx.Buildpack.Path, "resources", "custom-activity-correlation.xml"), []byte{}, 0644)).
			To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(ctx.Buildpack.Path, "resources", "log4j2.xml"), []byte{}, 0644)).
			To(Succeed())

		agentDep := libpak.BuildpackDependency{
			ID:     "appdynamics-java",
			URI:    "https://localhost/stub-appdynamics-agent.zip",
			SHA256: "ee23306ce5f7086219c1876652ed323970ebc249f21d1c79b737ac1120284bbf",
		}
		externalConfigurationDep := libpak.BuildpackDependency{
			ID:     "appdynamics-external-configuration",
			URI:    "https://localhost/stub-external-configuration.tar.gz",
			SHA256: "22e708cfd301430cbcf8d1c2289503d8288d50df519ff4db7cca0ff9fe83c324",
		}
		dc := libpak.DependencyCache{CachePath: "testdata"}

		j, bomEntries := appd.NewJavaAgent(ctx.Buildpack.Path, agentDep, libpak.ConfigurationResolver{}, &externalConfigurationDep, dc)
		Expect(bomEntries).To(HaveLen(2))
		Expect(bomEntries[0].Name).To(Equal("appdynamics-java"))
		Expect(bomEntries[0].Metadata["layer"]).To(Equal("appdynamics-java"))
		Expect(bomEntries[0].Launch).To(BeTrue())
		Expect(bomEntries[0].Build).To(BeFalse())
		Expect(bomEntries[1].Name).To(Equal("appdynamics-external-configuration"))
		Expect(bomEntries[1].Metadata["layer"]).To(Equal("appdynamics-java"))
		Expect(bomEntries[1].Launch).To(BeTrue())
		Expect(bomEntries[1].Build).To(BeFalse())

		layer, err := ctx.Layers.Layer("test-layer")
		Expect(err).NotTo(HaveOccurred())

		layer, err = j.Contribute(layer)
		Expect(err).NotTo(HaveOccurred())

		Expect(filepath.Join(layer.Path, "ver4.5.7.25056", "fixture-marker")).To(BeARegularFile())
	})

	context("$BP_TOMCAT_EXT_CONF_STRIP", func() {
		it.Before(func() {
			Expect(os.Setenv("BP_APPD_EXT_CONF_STRIP", "1")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("BP_APPD_EXT_CONF_STRIP")).To(Succeed())
		})

		it("contributes external configuration with directory", func() {
			Expect(os.MkdirAll(filepath.Join(ctx.Buildpack.Path, "resources"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(ctx.Buildpack.Path, "resources", "app-agent-config.xml"), []byte{}, 0644)).
				To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(ctx.Buildpack.Path, "resources", "custom-activity-correlation.xml"), []byte{}, 0644)).
				To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(ctx.Buildpack.Path, "resources", "log4j2.xml"), []byte{}, 0644)).
				To(Succeed())

			agentDep := libpak.BuildpackDependency{
				ID:     "appdynamics-java",
				URI:    "https://localhost/stub-appdynamics-agent.zip",
				SHA256: "ee23306ce5f7086219c1876652ed323970ebc249f21d1c79b737ac1120284bbf",
			}
			externalConfigurationDep := libpak.BuildpackDependency{
				ID:     "appdynamics-external-configuration",
				URI:    "https://localhost/stub-external-configuration-with-directory.tar.gz",
				SHA256: "060818cbcdc2008563f0f9e2428ecf4a199a5821c5b8b1dcd11a67666c1e2cd6",
			}
			dc := libpak.DependencyCache{CachePath: "testdata"}

			j, bomEntries := appd.NewJavaAgent(ctx.Buildpack.Path, agentDep, libpak.ConfigurationResolver{}, &externalConfigurationDep, dc)
			Expect(bomEntries).To(HaveLen(2))
			Expect(bomEntries[0].Name).To(Equal("appdynamics-java"))
			Expect(bomEntries[0].Metadata["layer"]).To(Equal("appdynamics-java"))
			Expect(bomEntries[0].Launch).To(BeTrue())
			Expect(bomEntries[0].Build).To(BeFalse())
			Expect(bomEntries[1].Name).To(Equal("appdynamics-external-configuration"))
			Expect(bomEntries[1].Metadata["layer"]).To(Equal("appdynamics-java"))
			Expect(bomEntries[1].Launch).To(BeTrue())
			Expect(bomEntries[1].Build).To(BeFalse())

			layer, err := ctx.Layers.Layer("test-layer")
			Expect(err).NotTo(HaveOccurred())

			layer, err = j.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			Expect(filepath.Join(layer.Path, "ver4.5.7.25056", "fixture-marker")).To(BeARegularFile())
		})

	})

}
