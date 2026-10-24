// Copyright 2018-2026 the original author or authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/Masterminds/semver/v3"
	"github.com/paketo-buildpacks/libdependency/buildpack_config"
	"github.com/paketo-buildpacks/libdependency/retrieve"
	"github.com/paketo-buildpacks/libdependency/upstream"
	"github.com/paketo-buildpacks/libdependency/versionology"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"gopkg.in/yaml.v3"
)

const licenseURI = "https://docs.appdynamics.com/display/PRO45/License+Entitlements+and+Restrictions"

var (
	versionPattern = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)_(.*)$`)
	buildPattern   = regexp.MustCompile(`(\d+)$`)
)

type dependency struct {
	id         string
	name       string
	purl       string
	root       string
	cpeVendor  string
	cpeProduct string
	arch       string
}

var dependencies = []dependency{
	{
		id:         "appdynamics-java",
		name:       "AppDynamics Java Agent",
		purl:       "pkg:generic/appdynamics-java-agent",
		root:       "https://download.run.pivotal.io/appdynamics",
		cpeVendor:  "appdynamics",
		cpeProduct: "java-agent",
	},
	{
		id:         "appdynamics-php",
		name:       "AppDynamics PHP Agent",
		purl:       "pkg:generic/appdynamics-php-agent",
		root:       "https://download.run.pivotal.io/appdynamics-php",
		cpeVendor:  "appdynamics",
		cpeProduct: "php-agent",
		arch:       "amd64",
	},
}

type version struct {
	version *semver.Version
	build   string
	uri     string
}

func (v version) Version() *semver.Version {
	return v.version
}

func main() {
	buildpackTomlPath, output := retrieve.FetchArgs()

	config, err := buildpack_config.ParseBuildpackToml(buildpackTomlPath)
	if err != nil {
		panic(err)
	}

	var metadata []versionology.Dependency
	for _, dep := range dependencies {
		newVersions, err := retrieve.GetNewVersionsForId(dep.id, config, versionsFunc(dep.root))
		if err != nil {
			panic(err)
		}

		metadata = append(metadata, retrieve.GenerateAllMetadata(newVersions, generateMetadata(dep))...)
	}

	content, err := json.Marshal(metadata)
	if err != nil {
		panic(fmt.Errorf("unable to marshal metadata\n%w", err))
	}

	if err := os.WriteFile(output, content, 0o644); err != nil {
		panic(fmt.Errorf("cannot write to %s: %w", output, err))
	}
	fmt.Printf("Wrote metadata to %s\n", output)
}

func versionsFunc(root string) retrieve.GetAllVersionsFunc {
	return func() (versionology.VersionFetcherArray, error) {
		return getAllVersions(root)
	}
}

func getAllVersions(root string) (versionology.VersionFetcherArray, error) {
	indexURL := fmt.Sprintf("%s/index.yml", root)

	resp, err := http.Get(indexURL)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch %s\n%w", indexURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unable to fetch %s: status code %d", indexURL, resp.StatusCode)
	}

	raw := map[string]string{}
	if err := yaml.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("unable to parse %s\n%w", indexURL, err)
	}

	latest := map[string]version{}
	for key, uri := range raw {
		m := versionPattern.FindStringSubmatch(key)
		if m == nil {
			continue
		}

		versionString := fmt.Sprintf("%s.%s.%s", m[1], m[2], m[3])
		v, err := semver.NewVersion(versionString)
		if err != nil {
			fmt.Printf("Skipping %s: unable to parse version\n", versionString)
			continue
		}

		candidate := version{version: v, build: m[4], uri: uri}
		if existing, ok := latest[versionString]; !ok || buildLess(existing.build, candidate.build) {
			latest[versionString] = candidate
		}
	}

	var versions versionology.VersionFetcherArray
	for _, v := range latest {
		versions = append(versions, v)
	}

	return versions, nil
}

func generateMetadata(dep dependency) retrieve.GenerateMetadataFunc {
	return func(versionFetcher versionology.VersionFetcher) ([]versionology.Dependency, error) {
		v, ok := versionFetcher.(version)
		if !ok {
			return nil, fmt.Errorf("unexpected version type %T", versionFetcher)
		}

		versionString := v.version.String()

		checksum, err := upstream.GetSHA256OfRemoteFile(v.uri)
		if err != nil {
			return nil, fmt.Errorf("unable to checksum %s\n%w", v.uri, err)
		}

		purl := fmt.Sprintf("%s@%s", dep.purl, versionString)
		if dep.arch != "" {
			purl = fmt.Sprintf("%s?arch=%s", purl, dep.arch)
		}

		dependency := cargo.ConfigMetadataDependency{
			Checksum: fmt.Sprintf("sha256:%s", checksum),
			CPE:      fmt.Sprintf("cpe:2.3:a:%s:%s:%s:*:*:*:*:*:*:*", dep.cpeVendor, dep.cpeProduct, versionString),
			ID:       dep.id,
			Licenses: []interface{}{
				map[string]string{
					"uri": licenseURI,
				},
			},
			Name:    dep.name,
			PURL:    purl,
			SHA256:  checksum,
			Stacks:  []string{"*"},
			URI:     v.uri,
			Version: versionString,
		}

		return versionology.NewDependencyArray(dependency, "")
	}
}

// buildLess reports whether build a is older than build b. Builds are numeric
// strings; compare them as integers with a string fallback.
func buildLess(a, b string) bool {
	an := buildPattern.FindString(a)
	bn := buildPattern.FindString(b)
	if an == "" || bn == "" {
		return a < b
	}

	ai, err := strconv.Atoi(an)
	if err != nil {
		return a < b
	}
	bi, err := strconv.Atoi(bn)
	if err != nil {
		return a < b
	}

	return ai < bi
}
