/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"
	"knative.dev/test-infra/shared/common"
)

const (
	// clusterConfigFile is the config file we need to put under the benchmark folder, if we want to config the cluster
	// that runs the benchmark, it must follow the scheme defined as GKECluster here.
	clusterConfigFile = "cluster.yaml"

	// These default settings will be used for configuring the cluster, if not specified in cluster.yaml.
	defaultLocation  = "us-central1"
	defaultNodeCount = 1
	defaultNodeType  = "n1-standard-4"
	defaultAddons    = "HorizontalPodAutoscaling,HttpLoadBalancing"
)

// backupLocations are used in retrying cluster creation, if stockout happens in one location.
// TODO(chizhg): it's currently not used, use it in the cluster creation retry logic.
var backupLocations = []string{"us-west1", "us-west2", "us-east1"}

// GKECluster saves the config information for the GKE cluster
type GKECluster struct {
	Config ClusterConfig `yaml:"GKECluster,omitempty"`
}

// ClusterConfig is config for the cluster
type ClusterConfig struct {
	Location  string `yaml:"location,omitempty"`
	NodeCount int64  `yaml:"nodeCount,omitempty"`
	NodeType  string `yaml:"nodeType,omitempty"`
	Addons    string `yaml:"addons,omitempty"`
}

// benchmarkNames returns names of the benchmarks.
//
// We put all benchmarks under the benchmarkRoot folder, one subfolder represents one benchmark,
// here we returns all subfolder names of the root folder.
func benchmarkNames(benchmarkRoot string) ([]string, error) {
	names := make([]string, 0)
	dirs, err := ioutil.ReadDir(benchmarkRoot)
	if err != nil {
		return names, fmt.Errorf("failed to list all benchmarks under %q: %v", benchmarkRoot, err)
	}

	for _, dir := range dirs {
		names = append(names, dir.Name())
	}
	return names, nil
}

// benchmarkClusters returns the cluster configs for all benchmarks.
func benchmarkClusters(repo, benchmarkRoot string) (map[string]ClusterConfig, error) {
	// clusters is a map of cluster configs
	// key is the cluster name, value is the cluster config
	clusters := make(map[string]ClusterConfig)
	benchmarkNames, err := benchmarkNames(benchmarkRoot)
	if err != nil {
		return clusters, err
	}

	for _, benchmarkName := range benchmarkNames {
		clusterConfig := clusterConfigForBenchmark(benchmarkRoot, benchmarkName)
		clusterName := clusterNameForBenchmark(repo, benchmarkName)
		clusters[clusterName] = clusterConfig
	}

	return clusters, nil
}

// clusterConfigForBenchmark returns the cluster config for the given benchmark.
//
// Under each benchmark folder, we can put a cluster.yaml file that follows the scheme we define
// in ClusterConfig struct, in which we specify configuration of the cluster that we use to run the benchmark.
// If there is no such config file, or the config file is malformed, default config will be used.
func clusterConfigForBenchmark(benchmarkRoot, benchmarkName string) ClusterConfig {
	gkeCluster := GKECluster{
		Config: ClusterConfig{
			Location:  defaultLocation,
			NodeCount: defaultNodeCount,
			NodeType:  defaultNodeType,
			Addons:    defaultAddons,
		},
	}

	configFile := filepath.Join(benchmarkRoot, benchmarkName, clusterConfigFile)
	if common.FileExists(configFile) {
		contents, err := ioutil.ReadFile(configFile)
		if err == nil {
			if err := yaml.Unmarshal(contents, &gkeCluster); err != nil {
				log.Printf("Failed to parse the config file %q, default config will be used", configFile)
			}
		} else {
			log.Printf("Failed to read the config file %q, default config will be used", configFile)
		}
	}

	return gkeCluster.Config
}

// clusterNameForBenchmark prepends repo name to the benchmark name, and use it as the cluster name.
func clusterNameForBenchmark(benchmarkName, repo string) string {
	return fmt.Sprintf("%s-%s", repo, benchmarkName)
}

// benchmarkNameForCluster removes repo name prefix from the cluster name, to get the real benchmark name.
// If the cluster does not belong to the given repo, return an empty string.
func benchmarkNameForCluster(clusterName, repo string) string {
	if !clusterBelongsToRepo(clusterName, repo) {
		return ""
	}
	return strings.TrimPrefix(clusterName, repo+"-")
}

// clusterBelongsToRepo determines if the cluster belongs to the repo, by checking if it has the repo prefix.
func clusterBelongsToRepo(clusterName, repo string) bool {
	return strings.HasPrefix(clusterName, repo+"-")
}
