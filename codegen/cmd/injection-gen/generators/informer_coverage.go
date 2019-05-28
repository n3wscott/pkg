/*
Copyright 2016 The Kubernetes Authors.

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

package generators

import (
	"io"
	"k8s.io/gengo/generator"
	"k8s.io/gengo/namer"
	"k8s.io/gengo/types"

	"k8s.io/klog"
)

// injectionTestGenerator produces a test file of informer injection for a given type.
type injectionTestGenerator struct {
	generator.DefaultGen
	typeToGenerate *types.Type
	imports        namer.ImportTracker
}

var _ generator.Generator = &injectionTestGenerator{}

func (g *injectionTestGenerator) Filter(c *generator.Context, t *types.Type) bool {
	return t == g.typeToGenerate
}

func (g *injectionTestGenerator) Namers(c *generator.Context) namer.NameSystems {
	return namer.NameSystems{}
}

func (g *injectionTestGenerator) Imports(c *generator.Context) (imports []string) {
	imports = append(imports, g.imports.ImportLines()...)
	return
}

func (g *injectionTestGenerator) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
	sw := generator.NewSnippetWriter(w, c, "{{", "}}")

	klog.V(5).Infof("processing type %v", t)

	m := map[string]interface{}{}

	sw.Do(injectionInformerTest, m)

	return sw.Error()
}

var injectionInformerTest = `
func TestRegistration(t *testing.T) {
	ctx := context.Background()

	// Get before registration
	if empty := Get(ctx); empty != nil {
		t.Errorf("Unexpected informer: %v", empty)
	}

	// Check how many informers have registered.
	inffs := injection.Default.GetInformers()
	if want, got := 1, len(inffs); want != got {
		t.Errorf("GetInformers() = %d, wanted %d", want, got)
	}

	// Setup the informers.
	var infs []controller.Informer
	ctx, infs = injection.Default.SetupInformers(ctx, &rest.Config{})

	// We should see that a single informer was set up.
	if want, got := 1, len(infs); want != got {
		t.Errorf("SetupInformers() = %d, wanted %d", want, got)
	}

	// Get our informer from the context.
	if inf := Get(ctx); inf == nil {
		t.Error("Get() = nil, wanted non-nil")
	}
}
`
