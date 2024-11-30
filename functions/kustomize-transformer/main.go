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

	"github.com/kuberik/kustomize-transformer-krm-fn/pkg/annotations"
	kuberik_filesys "github.com/kuberik/kustomize-transformer-krm-fn/pkg/filesys"
)

func transform(rl *fn.ResourceList) (bool, error) {
	memFS := filesys.MakeFsInMemory()
	fs, err := kuberik_filesys.NewSandboxFS(memFS, "")
	if err != nil {
		return false, err
	}

	kustomization := types.Kustomization{}
	for i, r := range rl.Items {
		filename := fmt.Sprintf("%d.yaml", i)
		if err := memFS.WriteFile(filename, []byte(r.String())); err != nil {
			return false, err
		}
		kustomization.Resources = append(kustomization.Resources, filename)
	}

	functionDir := "function"
	kustomizationDir := path.Join(functionDir, rl.FunctionConfig.GetAnnotation(annotations.KustomizationPathAnnotation))
	if err := memFS.WriteFile(path.Join(kustomizationDir, konfig.DefaultKustomizationFileName()), []byte(rl.FunctionConfig.String())); err != nil {
		return false, err
	}

	for key, value := range rl.FunctionConfig.GetAnnotations() {
		if !strings.HasPrefix(key, annotations.FileAnnotationPrefix) {
			continue
		}
		if err := memFS.WriteFile(
			path.Join(functionDir, strings.TrimPrefix(key, annotations.FileAnnotationPrefix)),
			[]byte(value),
		); err != nil {
			return false, err
		}
	}

	switch rl.FunctionConfig.GetKind() {
	case "Kustomization":
		kustomization.Resources = append(kustomization.Resources, kustomizationDir)
	}
	kustomizationContent, err := yaml.Marshal(kustomization)
	if err != nil {
		return false, err
	}
	if err := memFS.WriteFile(konfig.DefaultKustomizationFileName(), kustomizationContent); err != nil {
		return false, err
	}

	options := krusty.MakeDefaultOptions()
	options.PluginConfig.HelmConfig.Enabled = true
	options.PluginConfig.HelmConfig.Command = "helm"
	k := krusty.MakeKustomizer(options)

	m, err := k.Run(fs, ".")
	if err != nil {
		return false, err
	}
	rl.Items = fn.KubeObjects{}
	for _, r := range m.Resources() {
		ko, err := fn.ParseKubeObject([]byte(r.MustYaml()))
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
