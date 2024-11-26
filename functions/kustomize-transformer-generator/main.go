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
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/types"
)

var version string

func fileNameAnnotation(fileName string) string {
	return fmt.Sprintf("file.kustomize.kuberik.io/%s", fileName)
}

func generate(rl *fn.ResourceList) (bool, error) {
	resourcesDir := os.Getenv("RESOURCES_DIR")
	if resourcesDir == "" {
		resourcesDir = "/tmp"
	}
	kustomizationPath, _, _ := rl.FunctionConfig.NestedString("path")
	files, err := filepath.Glob(path.Join(resourcesDir, kustomizationPath, "*"))
	if err != nil {
		return false, err
	}
	fileAnnotations := make(map[string]string)
	kustomizationFile := ""
	for _, file := range files {
		fileInfo, err := os.Stat(file)
		if err != nil {
			return false, err
		}
		if !fileInfo.Mode().IsRegular() {
			continue
		}
		fileName := filepath.Base(file)
		if slices.Contains(konfig.RecognizedKustomizationFileNames(), fileName) {
			if kustomizationFile != "" {
				return false, fmt.Errorf("multiple kustomization files found in %s", kustomizationPath)
			}
			kustomizationFile = file
		}
		contents, err := os.ReadFile(file)
		if err != nil {
			return false, err
		}
		fileAnnotations[fileNameAnnotation(fileName)] = string(contents)
	}
	if kustomizationFile == "" {
		return false, fmt.Errorf("kustomization file not found in %s", kustomizationPath)
	}
	kustomization := &types.Kustomization{}
	if err := kustomization.Unmarshal([]byte(fileAnnotations[fileNameAnnotation(filepath.Base(kustomizationFile))])); err != nil {
		return false, err
	}
	delete(fileAnnotations, fileNameAnnotation(filepath.Base(kustomizationFile)))

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
	}
	maps.Copy(kustomization.MetaData.Annotations, fileAnnotations)
	kustomization.MetaData.Annotations["config.kubernetes.io/function"] = strings.TrimSpace(fmt.Sprintf(`
container:
  image: ghcr.io/kuberik/kpt-fn/kustomize-transformer:%s
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
