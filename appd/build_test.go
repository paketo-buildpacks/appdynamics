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
	"os"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/appdynamics/v4/appd"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx libcnb.BuildContext
	)

	it("contributes Java agent API <= 0.6", func() {
		ctx.Plan.Entries = append(ctx.Plan.Entries, libcnb.BuildpackPlanEntry{Name: "appdynamics-java"})
		ctx.Buildpack.Metadata = map[string]interface{}{
			"dependencies": []map[string]interface{}{
				{
					"id":      "appdynamics-java",
					"version": "1.1.1",
					"stacks":  []interface{}{"test-stack-id"},
				},
			},
		}
		ctx.Buildpack.API = "0.6"
		ctx.StackID = "test-stack-id"

		result, err := appd.Build{}.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(2))
		Expect(result.Layers[0].Name()).To(Equal("appdynamics-java"))
		Expect(result.Layers[1].Name()).To(Equal("helper"))
		Expect(result.Layers[1].(libpak.HelperLayerContributor).Names).To(Equal([]string{"properties"}))
		Expect(result.BOM.Entries).To(HaveLen(2))
		Expect(result.BOM.Entries[0].Name).To(Equal("appdynamics-java"))
		Expect(result.BOM.Entries[1].Name).To(Equal("helper"))
	})

	it("contributes Java agent API >= 0.7", func() {
		ctx.Plan.Entries = append(ctx.Plan.Entries, libcnb.BuildpackPlanEntry{Name: "appdynamics-java"})
		ctx.Buildpack.Metadata = map[string]interface{}{
			"dependencies": []map[string]interface{}{
				{
					"id":      "appdynamics-java",
					"version": "1.1.1",
					"stacks":  []interface{}{"test-stack-id"},
					"cpes":    []interface{}{"cpe:2.3:a:appdynamics:java-agent:1.1.1:*:*:*:*:*:*:*"},
					"purl":    "pkg:generic/appdynamics-java-agent@1.1.1?arch=amd64",
				},
			},
		}
		ctx.Buildpack.API = "0.7"
		ctx.StackID = "test-stack-id"

		result, err := appd.Build{}.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(2))
		Expect(result.Layers[0].Name()).To(Equal("appdynamics-java"))
		Expect(result.Layers[1].Name()).To(Equal("helper"))
		Expect(result.Layers[1].(libpak.HelperLayerContributor).Names).To(Equal([]string{"properties"}))
		Expect(result.BOM.Entries).To(HaveLen(2))
		Expect(result.BOM.Entries[0].Name).To(Equal("appdynamics-java"))
		Expect(result.BOM.Entries[1].Name).To(Equal("helper"))
	})

	context("$BP_APPD_EXT_CONF_URI", func() {
		it.Before(func() {
			Expect(os.Setenv("BP_APPD_EXT_CONF_SHA256", "test-sha256")).To(Succeed())
			Expect(os.Setenv("BP_APPD_EXT_CONF_URI", "test-uri")).To(Succeed())
			Expect(os.Setenv("BP_APPD_EXT_CONF_VERSION", "test-version")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("BP_APPD_EXT_CONF_SHA256")).To(Succeed())
			Expect(os.Unsetenv("BP_APPD_EXT_CONF_URI")).To(Succeed())
			Expect(os.Unsetenv("BP_APPD_EXT_CONF_VERSION")).To(Succeed())
		})

		it("contributes extensions when $BP_APPD_EXT_CONF_URI is set API <= 0.6", func() {
			ctx.Plan.Entries = append(ctx.Plan.Entries, libcnb.BuildpackPlanEntry{Name: "appdynamics-java"})
			ctx.Buildpack.Metadata = map[string]interface{}{
				"dependencies": []map[string]interface{}{
					{
						"id":      "appdynamics-java",
						"version": "1.1.1",
						"stacks":  []interface{}{"test-stack-id"},
					},
				},
			}
			ctx.Buildpack.API = "0.6"
			ctx.StackID = "test-stack-id"

			result, err := appd.Build{}.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(2))
			Expect(result.Layers[0].(appd.JavaAgent).ExternalConfigurationDependency).To(Equal(&libpak.BuildpackDependency{
				ID:      "appdynamics-external-configuration",
				Name:    "AppDynamics External Configuration",
				Version: "test-version",
				URI:     "test-uri",
				SHA256:  "test-sha256",
				Stacks:  []string{ctx.StackID},
				CPEs:    []string{"cpe:2.3:a:appdynamics:external-configuration:test-version:*:*:*:*:*:*:*"},
				PURL:    "pkg:generic/appdynamics-external-configuration@test-version",
			}))
			Expect(result.Layers[1].(libpak.HelperLayerContributor).Names).To(Equal([]string{"properties"}))

			Expect(result.BOM.Entries).To(HaveLen(3))
			Expect(result.BOM.Entries[0].Name).To(Equal("appdynamics-java"))
			Expect(result.BOM.Entries[1].Name).To(Equal("appdynamics-external-configuration"))
			Expect(result.BOM.Entries[2].Name).To(Equal("helper"))
		})

		it("contributes extensions when $BP_APPD_EXT_CONF_URI is set API >= 0.7", func() {
			ctx.Plan.Entries = append(ctx.Plan.Entries, libcnb.BuildpackPlanEntry{Name: "appdynamics-java"})
			ctx.Buildpack.Metadata = map[string]interface{}{
				"dependencies": []map[string]interface{}{
					{
						"id":      "appdynamics-java",
						"version": "1.1.1",
						"stacks":  []interface{}{"test-stack-id"},
						"cpes":    []interface{}{"cpe:2.3:a:appdynamics:java-agent:1.1.1:*:*:*:*:*:*:*"},
						"purl":    "pkg:generic/appdynamics-java-agent@1.1.1?arch=amd64",
					},
				},
			}
			ctx.Buildpack.API = "0.7"
			ctx.StackID = "test-stack-id"

			result, err := appd.Build{}.Build(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(2))
			Expect(result.Layers[0].(appd.JavaAgent).ExternalConfigurationDependency).To(Equal(&libpak.BuildpackDependency{
				ID:      "appdynamics-external-configuration",
				Name:    "AppDynamics External Configuration",
				Version: "test-version",
				URI:     "test-uri",
				SHA256:  "test-sha256",
				Stacks:  []string{ctx.StackID},
				CPEs:    []string{"cpe:2.3:a:appdynamics:external-configuration:test-version:*:*:*:*:*:*:*"},
				PURL:    "pkg:generic/appdynamics-external-configuration@test-version",
			}))
			Expect(result.Layers[1].(libpak.HelperLayerContributor).Names).To(Equal([]string{"properties"}))

			Expect(result.BOM.Entries).To(HaveLen(3))
			Expect(result.BOM.Entries[0].Name).To(Equal("appdynamics-java"))
			Expect(result.BOM.Entries[1].Name).To(Equal("appdynamics-external-configuration"))
			Expect(result.BOM.Entries[2].Name).To(Equal("helper"))
		})
	})

	it("contributes NodeJS agent API <= 0.6", func() {
		ctx.Plan.Entries = append(ctx.Plan.Entries, libcnb.BuildpackPlanEntry{Name: "appdynamics-nodejs"})
		ctx.Buildpack.Metadata = map[string]interface{}{
			"dependencies": []map[string]interface{}{
				{
					"id":      "appdynamics-nodejs",
					"version": "1.1.1",
					"stacks":  []interface{}{"test-stack-id"},
				},
			},
		}
		ctx.Buildpack.API = "0.6"
		ctx.StackID = "test-stack-id"

		result, err := appd.Build{}.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(2))
		Expect(result.Layers[0].Name()).To(Equal("appdynamics-nodejs"))
		Expect(result.Layers[1].(libpak.HelperLayerContributor).Names).To(Equal([]string{"properties"}))

		Expect(result.BOM.Entries).To(HaveLen(2))
		Expect(result.BOM.Entries[0].Name).To(Equal("appdynamics-nodejs"))
		Expect(result.BOM.Entries[1].Name).To(Equal("helper"))
	})

	it("contributes NodeJS agent API >= 0.7", func() {
		ctx.Plan.Entries = append(ctx.Plan.Entries, libcnb.BuildpackPlanEntry{Name: "appdynamics-nodejs"})
		ctx.Buildpack.Metadata = map[string]interface{}{
			"dependencies": []map[string]interface{}{
				{
					"id":      "appdynamics-nodejs",
					"version": "1.1.1",
					"stacks":  []interface{}{"test-stack-id"},
					"cpes":    []interface{}{"cpe:2.3:a:appdynamics:nodejs-agent:1.1.1:*:*:*:*:*:*:*"},
					"purl":    "pkg:generic/appdynamics-nodejs-agent@1.1.1?arch=amd64",
				},
			},
		}
		ctx.Buildpack.API = "0.7"
		ctx.StackID = "test-stack-id"

		result, err := appd.Build{}.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(2))
		Expect(result.Layers[0].Name()).To(Equal("appdynamics-nodejs"))
		Expect(result.Layers[1].(libpak.HelperLayerContributor).Names).To(Equal([]string{"properties"}))

		Expect(result.BOM.Entries).To(HaveLen(2))
		Expect(result.BOM.Entries[0].Name).To(Equal("appdynamics-nodejs"))
		Expect(result.BOM.Entries[1].Name).To(Equal("helper"))
	})

	it("contributes PHP agent API <= 0.6", func() {
		ctx.Plan.Entries = append(ctx.Plan.Entries, libcnb.BuildpackPlanEntry{Name: "appdynamics-php"})
		ctx.Buildpack.Metadata = map[string]interface{}{
			"dependencies": []map[string]interface{}{
				{
					"id":      "appdynamics-php",
					"version": "1.1.1",
					"stacks":  []interface{}{"test-stack-id"},
				},
			},
		}
		ctx.Buildpack.API = "0.6"
		ctx.StackID = "test-stack-id"

		result, err := appd.Build{}.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(2))
		Expect(result.Layers[0].Name()).To(Equal("appdynamics-php"))
		Expect(result.Layers[1].Name()).To(Equal("helper"))
		Expect(result.Layers[1].(libpak.HelperLayerContributor).Names).To(Equal([]string{"properties"}))

		Expect(result.BOM.Entries).To(HaveLen(2))
		Expect(result.BOM.Entries[0].Name).To(Equal("appdynamics-php"))
		Expect(result.BOM.Entries[1].Name).To(Equal("helper"))
	})

	it("contributes PHP agent API >= 0.7", func() {
		ctx.Plan.Entries = append(ctx.Plan.Entries, libcnb.BuildpackPlanEntry{Name: "appdynamics-php"})
		ctx.Buildpack.Metadata = map[string]interface{}{
			"dependencies": []map[string]interface{}{
				{
					"id":      "appdynamics-php",
					"version": "1.1.1",
					"stacks":  []interface{}{"test-stack-id"},
					"cpes":    []interface{}{"cpe:2.3:a:appdynamics:php-agent:1.1.1:*:*:*:*:*:*:*"},
					"purl":    "pkg:generic/appdynamics-php-agent@1.1.1?arch=amd64",
				},
			},
		}
		ctx.Buildpack.API = "0.7"
		ctx.StackID = "test-stack-id"

		result, err := appd.Build{}.Build(ctx)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(2))
		Expect(result.Layers[0].Name()).To(Equal("appdynamics-php"))
		Expect(result.Layers[1].Name()).To(Equal("helper"))
		Expect(result.Layers[1].(libpak.HelperLayerContributor).Names).To(Equal([]string{"properties"}))

		Expect(result.BOM.Entries).To(HaveLen(2))
		Expect(result.BOM.Entries[0].Name).To(Equal("appdynamics-php"))
		Expect(result.BOM.Entries[1].Name).To(Equal("helper"))
	})

}
