/*
Copyright 2019 The Knative Authors.

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

// factoryTestGenerator produces a file of factory injection of a given type.
type factoryGenerator struct {
	generator.DefaultGen
	outputPackage                string
	imports                      namer.ImportTracker
	cachingClientSetPackage      string
	sharedInformerFactoryPackage string
	filtered                     bool
}

var _ generator.Generator = &factoryGenerator{}

func (g *factoryGenerator) Filter(c *generator.Context, t *types.Type) bool {
	if !g.filtered {
		g.filtered = true
		return true
	}
	return false
}

func (g *factoryGenerator) Namers(c *generator.Context) namer.NameSystems {
	return namer.NameSystems{
		"raw": namer.NewRawNamer(g.outputPackage, g.imports),
	}
}

func (g *factoryGenerator) Imports(c *generator.Context) (imports []string) {
	imports = append(imports, g.imports.ImportLines()...)
	return
}

func (g *factoryGenerator) GenerateType(c *generator.Context, t *types.Type, w io.Writer) error {
	sw := generator.NewSnippetWriter(w, c, "{{", "}}")

	klog.V(5).Infof("processing type %v", t)

	m := map[string]interface{}{
		"cachingClientGet":                  c.Universe.Type(types.Name{Package: g.cachingClientSetPackage, Name: "Get"}),
		"informersNewSharedInformerFactory": c.Universe.Function(types.Name{Package: g.sharedInformerFactoryPackage, Name: "NewSharedInformerFactory"}),
		"informersSharedInformerFactory":    c.Universe.Function(types.Name{Package: g.sharedInformerFactoryPackage, Name: "SharedInformerFactory"}),
		"injectionRegisterInformerFactory":  c.Universe.Type(types.Name{Package: "github.com/knative/pkg/injection", Name: "Default.RegisterInformerFactory"}),
		"controllerGetResyncPeriod":         c.Universe.Type(types.Name{Package: "github.com/knative/pkg/controller", Name: "GetResyncPeriod"}),
	}

	sw.Do(injectionFactory, m)

	return sw.Error()
}

var injectionFactory = `
func init() {
	{{.injectionRegisterInformerFactory|raw}}(withInformerFactory)
}

// key is used as the key for associating information with a context.Context.
type Key struct{}

func withInformerFactory(ctx context.Context) context.Context {
	c := {{.cachingClientGet|raw}}(ctx)
	return context.WithValue(ctx, Key{},
		{{.informersNewSharedInformerFactory|raw}}(c, {{.controllerGetResyncPeriod|raw}}(ctx)))
}

// Get extracts the InformerFactory from the context.
func Get(ctx context.Context) {{.informersSharedInformerFactory|raw}} {
	return ctx.Value(Key{}).({{.informersSharedInformerFactory|raw}})
}
`
