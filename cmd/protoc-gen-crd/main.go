// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"strings"

	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"

	"istio.io/tools/cmd/protoc-gen-crd/pkg/protocgen"
	"istio.io/tools/pkg/protomodel"
)

// Breaks the comma-separated list of key=value pairs
// in the parameter string into an easy to use map.
func extractParams(parameter string) map[string]string {
	m := make(map[string]string)
	for _, p := range strings.Split(parameter, ",") {
		if p == "" {
			continue
		}

		if i := strings.Index(p, "="); i < 0 {
			m[p] = ""
		} else {
			m[p[0:i]] = p[i+1:]
		}
	}

	return m
}

func generate(request *plugin.CodeGeneratorRequest) (*plugin.CodeGeneratorResponse, error) {
	includeDescription := true
	enumAsIntOrString := false

	p := extractParams(request.GetParameter())
	for k, v := range p {
		if k == "include_description" {
			switch strings.ToLower(v) {
			case "true":
				includeDescription = true
			case "false":
				includeDescription = false
			default:
				return nil, fmt.Errorf("unknown value '%s' for include_description", v)
			}
		} else if k == "enum_as_int_or_string" {
			switch strings.ToLower(v) {
			case "true":
				enumAsIntOrString = true
			case "false":
				enumAsIntOrString = false
			default:
				return nil, fmt.Errorf("unknown value '%s' for enum_as_int_or_string", v)
			}
		} else {
			return nil, fmt.Errorf("unknown argument '%s' specified", k)
		}
	}

	m := protomodel.NewModel(request, false)

	legacyChannelFilesToGen := make(map[*protomodel.FileDescriptor]bool)
	standardChannelFilesToGen := make(map[*protomodel.FileDescriptor]bool)
	experimentalChannelFilesToGen := make(map[*protomodel.FileDescriptor]bool)

	for _, fileName := range request.FileToGenerate {
		fd := m.AllFilesByName[fileName]
		if fd == nil {
			return nil, fmt.Errorf("unable to find %s", request.FileToGenerate)
		} else if strings.HasSuffix(fd.GetPackage(), "v1") {
			standardChannelFilesToGen[fd] = true
			log.Println("it is standard: ", fd)
		} else if strings.HasSuffix(fd.GetPackage(), "v1alpha1") {
			experimentalChannelFilesToGen[fd] = true
			log.Println("it is experimental: ", fd)
		}
		// Legacy channel will have all files that are in standard and experimental
		legacyChannelFilesToGen[fd] = true
	}

	channelOutput := make(map[string]map[*protomodel.FileDescriptor]bool)
	channelOutput["kubernetes/legacy.gen.yaml"] = legacyChannelFilesToGen
	channelOutput["kubernetes/standard.gen.yaml"] = standardChannelFilesToGen
	channelOutput["kubernetes/exerimental.gen.yaml"] = experimentalChannelFilesToGen

	descriptionConfiguration := &DescriptionConfiguration{
		IncludeDescriptionInSchema: includeDescription,
	}

	g := newOpenAPIGenerator(
		m,
		descriptionConfiguration,
		enumAsIntOrString)

	for outputFileName, files := range channelOutput {
		// TODO (whgriffi): fix the return to generate multiple files. At this time only the first file in the list is returned
		return g.generateOutput(files, outputFileName)
	}

	return nil, nil
}

func main() {
	// TODO(whgriffi): may need to loop through the various channels here to generate multiple files as files aren't outputted by just running g.generateOutput(files, outputFileName)
	// The protocgen.Generate function is part of the protoc-gen-go plugin for the Protocol Buffers compiler (protoc).
	// This function handles the communication with protoc, including reading the CodeGeneratorRequest from protoc and writing the CodeGeneratorResponse back to protoc.
	// channels = make(map[string][]string) // map of channel name to acceptable API versions
	// channels["legacy"] = []string{"v1alpha1", "v1beta1", "v1"}
	// channels["standard"] = []string{"v1beta1", "v1"}
	// channels["experimental"] = []string{"v1alpha1"}

	// for _, channel := range channels {

	// }
	protocgen.Generate(generate)
}
