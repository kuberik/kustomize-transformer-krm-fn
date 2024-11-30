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
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"

	"github.com/kuberik/kustomize-transformer-krm-fn/pkg/annotations"
	kuberik_filesys "github.com/kuberik/kustomize-transformer-krm-fn/pkg/filesys"
)

var version string

func fileNameAnnotation(fileName string) string {
	return fmt.Sprintf("%s%s", annotations.FileAnnotationPrefix, fileName)
}

func findKustomizeFiles(rootPath string, relativeKustomizationPath string) ([]string, error) {
	// TODO: also add the same to both kustomize-transformer-generator and kustomize-transformer
	// TODO: check if options are the same as in kustomize cmd
	options := krusty.MakeDefaultOptions()
	options.PluginConfig.HelmConfig.Enabled = true
	options.PluginConfig.HelmConfig.Command = "helm"

	trackerFs := kuberik_filesys.NewFileSystemAccessTracker(filesys.MakeFsOnDisk())
	fs, err := kuberik_filesys.NewSandboxFS(trackerFs, rootPath)
	if err != nil {
		return nil, err
	}

	k := krusty.MakeKustomizer(options)
	_, err = k.Run(fs, relativeKustomizationPath)
	if err != nil {
		return nil, err
	}
	return trackerFs.AccessedFiles(), nil
}

func generate(rl *fn.ResourceList) (bool, error) {
	resourcesDir := os.Getenv("RESOURCES_DIR")
	if resourcesDir == "" {
		resourcesDir = "/tmp"
	}
	relativeKustomizationPath, _, _ := rl.FunctionConfig.NestedString("path")
	files, err := findKustomizeFiles(resourcesDir, relativeKustomizationPath)
	if err != nil {
		return false, err
	}
	fileAnnotations := make(map[string]string)
	kustomizationAnnotation := ""
	for _, file := range files {
		relPath, err := filepath.Rel(resourcesDir, file)
		if err != nil {
			return false, err
		}
		if !filepath.IsLocal(relPath) {
			continue
		}
		fileInfo, err := os.Stat(file)
		if err != nil {
			return false, err
		}
		if !fileInfo.Mode().IsRegular() {
			continue
		}
		fileName := filepath.Base(file)
		annotation := fileNameAnnotation(relPath)
		if slices.Contains(konfig.RecognizedKustomizationFileNames(), fileName) && filepath.Dir(relPath) == relativeKustomizationPath {
			if kustomizationAnnotation != "" {
				return false, fmt.Errorf("multiple kustomization files found in %s", relativeKustomizationPath)
			}
			kustomizationAnnotation = annotation
		}
		contents, err := os.ReadFile(file)
		if err != nil {
			return false, err
		}
		fileAnnotations[annotation] = string(contents)
	}
	if kustomizationAnnotation == "" {
		return false, fmt.Errorf("kustomization file not found in %s", relativeKustomizationPath)
	}
	kustomization := &types.Kustomization{}
	if err := kustomization.Unmarshal([]byte(fileAnnotations[kustomizationAnnotation])); err != nil {
		return false, err
	}
	delete(fileAnnotations, kustomizationAnnotation)

	// set type meta
	if kustomization.TypeMeta.Kind == "" {
		kustomization.TypeMeta.Kind = "Kustomization"
	}
	if kustomization.TypeMeta.APIVersion == "" {
		kustomization.TypeMeta.APIVersion = "kustomize.config.k8s.io/v1beta1"
	}

	// set name
	if kustomization.MetaData == nil {
		kustomization.MetaData = &types.ObjectMeta{}
	}
	kustomization.MetaData.Name = rl.FunctionConfig.GetName()

	// set annotations
	if kustomization.MetaData.Annotations == nil {
		kustomization.MetaData.Annotations = make(map[string]string)
		kustomization.MetaData.Annotations[annotations.KustomizationPathAnnotation] = relativeKustomizationPath
	}
	maps.Copy(kustomization.MetaData.Annotations, fileAnnotations)
	kustomization.MetaData.Annotations["config.kubernetes.io/function"] = strings.TrimSpace(fmt.Sprintf(`
container:
  image: ghcr.io/kuberik/kpt-fn/kustomize-transformer:%s
  network: true
`, version))

	generatedKustomizationTransformer, err := fn.NewFromTypedObject(kustomization)
	if err != nil {
		return false, err
	}
	rl.Items = append(rl.Items, generatedKustomizationTransformer)
	return true, nil
}

func main() {
	if err := fn.AsMain(fn.ResourceListProcessorFunc(generate)); err != nil {
		os.Exit(1)
	}
}
