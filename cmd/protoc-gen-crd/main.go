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
	"regexp"
	"strings"

	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"

	"istio.io/tools/cmd/protoc-gen-crd/pkg/protocgen"
	"istio.io/tools/pkg/protomodel"
)

var (
	experimentalRegex = regexp.MustCompile(`v[0-9]+(alpha|beta)[0-9]+$`)
	standardRegex     = regexp.MustCompile(`v[0-9]+$`)
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

	legacyChannelFilesToGen := make(map[*protomodel.FileDescriptor]struct{})
	standardChannelFilesToGen := make(map[*protomodel.FileDescriptor]struct{})
	experimentalChannelFilesToGen := make(map[*protomodel.FileDescriptor]struct{})

	for _, fileName := range request.FileToGenerate {
		fd := m.AllFilesByName[fileName]
		if fd == nil {
			return nil, fmt.Errorf("unable to find %s", request.FileToGenerate)
		}
		if standardRegex.MatchString(fd.GetPackage()) {
			standardChannelFilesToGen[fd] = true
			log.Println("it is standard: ", fd)
		} else if experimentalRegex.MatchString(fd.GetPackage()) {
			experimentalChannelFilesToGen[fd] = true
			log.Println("it is experimental: ", fd)
		}
		// Legacy channel will have all files that are in standard and experimental
		log.Println("This is also added to legacy channel: ", fd)
		legacyChannelFilesToGen[fd] = struct{}{}
	}

	channelOutput := make(map[string]map[*protomodel.FileDescriptor]bool)
	channelOutput["kubernetes/legacy.gen.yaml"] = legacyChannelFilesToGen
	channelOutput["kubernetes/standard.gen.yaml"] = standardChannelFilesToGen
	channelOutput["kubernetes/experimental.gen.yaml"] = experimentalChannelFilesToGen

	descriptionConfiguration := &DescriptionConfiguration{
		IncludeDescriptionInSchema: includeDescription,
	g := newOpenAPIGenerator(
		m,
		descriptionConfiguration,
		enumAsIntOrString)

	channels["kubernetes/legacy.gen.yaml"] = legacyChannelFilesToGen
	channels["kubernetes/experimental.gen.yaml"] = experimentalChannelFilesToGen
	channels["kubernetes/standard.gen.yaml"] = standardChannelFilesToGen
	return g.generateOutput(channels)
}

func main() {
	protocgen.Generate(generate)
}
