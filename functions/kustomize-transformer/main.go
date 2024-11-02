// Copyright 2022 Google LLC
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

package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const fileAnnotationPrefix = "file.kustomize.kuberik.io/"

func transform(rl *fn.ResourceList) (bool, error) {
	fs := filesys.MakeFsInMemory()

	kustomization := types.Kustomization{}
	for i, r := range rl.Items {
		filename := fmt.Sprintf("%d.yaml", i)
		if err := fs.WriteFile(filename, []byte(r.String())); err != nil {
			return false, err
		}
		kustomization.Resources = append(kustomization.Resources, filename)
	}

	functionDir := "function"
	if err := fs.WriteFile(path.Join(functionDir, konfig.DefaultKustomizationFileName()), []byte(rl.FunctionConfig.String())); err != nil {
		return false, err
	}
	kustomization.Resources = append(kustomization.Resources, functionDir)

	for key, value := range rl.FunctionConfig.GetAnnotations() {
		if !strings.HasPrefix(key, fileAnnotationPrefix) {
			continue
		}
		if err := fs.WriteFile(
			path.Join(functionDir, strings.TrimPrefix(key, fileAnnotationPrefix)),
			[]byte(value),
		); err != nil {
			return false, err
		}
	}

	switch rl.FunctionConfig.GetKind() {
	case "Kustomize":
		kustomization.Resources = append(kustomization.Resources, functionDir)
	}
	kustomizationContent, err := yaml.Marshal(kustomization)
	if err != nil {
		return false, err
	}
	if err := fs.WriteFile(konfig.DefaultKustomizationFileName(), kustomizationContent); err != nil {
		return false, err
	}

	k := krusty.MakeKustomizer(
		krusty.MakeDefaultOptions(),
	)

	m, err := k.Run(fs, ".")
	if err != nil {
		return false, err
	}
	rl.Items = fn.KubeObjects{}
	for _, r := range m.Resources() {
		o, err := r.Map()
		if err != nil {
			return false, err
		}
		ko, err := fn.NewFromTypedObject(o)
		if err != nil {
			return false, err
		}
		rl.Items = append(rl.Items, ko)
	}
	return true, nil
}

func main() {
	if err := fn.AsMain(fn.ResourceListProcessorFunc(transform)); err != nil {
		os.Exit(1)
	}
}
