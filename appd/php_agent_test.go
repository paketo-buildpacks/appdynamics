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
	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/libpak/effect/mocks"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/appdynamics/v4/appd"
)

func testPHPAgent(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		ctx      libcnb.BuildContext
		executor *mocks.Executor
	)

	it.Before(func() {
		var err error

		ctx.Layers.Path, err = ioutil.TempDir("", "php-agent-layers")
		Expect(err).NotTo(HaveOccurred())

		executor = &mocks.Executor{}
		executor.On("Execute", mock.Anything).Return(nil)
	})

	it.After(func() {
		Expect(os.RemoveAll(ctx.Layers.Path)).To(Succeed())
	})

	context("PHP_EXTENSION_DIR", func() {
		it.Before(func() {
			Expect(os.Setenv("PHP_EXTENSION_DIR", "test-extension-dir"))
		})

		it.After(func() {
			Expect(os.Unsetenv("PHP_EXTENSION_DIR"))
		})

		it("contributes PHP agent", func() {
			dep := libpak.BuildpackDependency{
				URI:    "https://localhost/stub-appdynamics-agent.tar.bz2",
				SHA256: "4918be522d0e00aa799d924266f00422ea059d7ee78177e1dde3335549433df7",
			}
			dc := libpak.DependencyCache{CachePath: "testdata"}

			j, _ := appd.NewPHPAgent(dep, dc)
			j.Executor = executor
			layer, err := ctx.Layers.Layer("test-layer")
			Expect(err).NotTo(HaveOccurred())

			Expect(os.MkdirAll(filepath.Join(layer.Path, "php.ini.d"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(layer.Path, "php.ini.d", "appdynamics_agent.ini"), []byte{}, 0644)).To(Succeed())

			layer, err = j.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			execution := executor.Calls[0].Arguments[0].(effect.Execution)
			Expect(execution.Command).To(Equal(filepath.Join(layer.Path, "install.sh")))
			Expect(execution.Args).To(Equal([]string{
				"--ignore-permissions",
				"--php-extension-dir=test-extension-dir",
				fmt.Sprintf("--php-ini-dir=%s", filepath.Join(layer.Path, "php.ini.d")),
				"--account-info=${APPDYNAMICS_AGENT_ACCOUNT_NAME}@${APPDYNAMICS_AGENT_ACCOUNT_ACCESS_KEY}",
				"${APPDYNAMICS_CONTROLLER_HOST_NAME}",
				"${APPDYNAMICS_CONTROLLER_PORT}",
				"${APPDYNAMICS_AGENT_APPLICATION_NAME}",
				"${APPDYNAMICS_AGENT_TIER_NAME}",
				"${APPDYNAMICS_AGENT_NODE_NAME}",
			}))

			Expect(layer.LaunchEnvironment["PHP_INI_SCAN_DIR.delim"]).To(Equal(string(os.PathListSeparator)))
			Expect(layer.LaunchEnvironment["PHP_INI_SCAN_DIR.prepend"]).To(Equal(filepath.Join(layer.Path, "php.ini.d")))
			Expect(ioutil.ReadFile(filepath.Join(layer.Path, "php.ini.d", "appdynamics_agent.ini"))).To(Equal([]byte(fmt.Sprintf(
				`
agent.controller.ssl.enabled = ${APPDYNAMICS_CONTROLLER_SSL_ENABLED}
`))))
		})
	})

}
