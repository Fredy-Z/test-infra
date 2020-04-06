/*
Copyright 2018 The Knative Authors

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
	gb "go/build"
	"log"
	"sort"
	"strings"
)

func CollectTransitiveImports(binaries []string) ([]string, error) {
	// Perform a simple DFS to collect the binaries' transitive dependencies.
	visited := make(map[string]struct{})
	for _, importpath := range binaries {
		g := &gobuild{moduleInfo()}

		if gb.IsLocalImport(importpath) {
			ip, err := g.qualifyLocalImport(importpath)
			if err != nil {
				return nil, err
			}
			importpath = ip
		}

		pkg, err := g.importPackage(importpath)
		if err != nil {
			return nil, err
		}
		if err := visit(g, pkg, visited); err != nil {
			return nil, err
		}
	}

	// Sort the dependencies deterministically.
	var list sort.StringSlice
	for ip := range visited {
		if !strings.Contains(ip, "/vendor/") {
			// Skip files outside of vendor
			continue
		}
		list = append(list, ip)
	}
	list.Sort()

	return list, nil
}

func visit(g *gobuild, pkg *gb.Package, visited map[string]struct{}) error {
	if _, ok := visited[pkg.Dir]; ok {
		return nil
	}
	visited[pkg.Dir] = struct{}{}
	log.Println(pkg)

	for _, ip := range pkg.Imports {
		if ip == "C" {
			// skip cgo
			continue
		}
		subpkg, err := g.importPackage(ip)
		if err != nil {
			return fmt.Errorf("%v\n -> %v", pkg.ImportPath, err)
		}
		if !strings.HasPrefix(subpkg.Dir, WorkingDir) {
			// Skip import paths outside of our workspace (std library)
			continue
		}
		if err := visit(g, subpkg, visited); err != nil {
			return fmt.Errorf("%v (%v)\n -> %v", pkg.ImportPath, pkg.Dir, err)
		}
	}
	return nil
}
