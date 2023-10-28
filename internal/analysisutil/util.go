package analysisutil

import (
	"go/types"

	"container/list"

	"github.com/gostaticanalysis/analysisutil"
	"golang.org/x/tools/go/analysis"
)

// copied and modified from https://github.com/gostaticanalysis/analysisutil/blob/ccfdecf515f47e636ba164ce0e5f26810eaf8747/types.go#L18
// ObjectOf returns types.Object by given name in the package.
func ObjectOf(pass *analysis.Pass, pkg, name string) types.Object {
	obj := analysisutil.LookupFromImports(pass.Pkg.Imports(), pkg, name)
	if obj != nil {
		return obj
	}
	if analysisutil.RemoveVendor(pass.Pkg.Path()) != analysisutil.RemoveVendor(pkg) {
		return nil
	}
	return pass.Pkg.Scope().Lookup(name)
}

func TypeOfBFS(pkg *types.Package, path, name string) types.Type {
	if name == "" {
		return nil
	}

	if name[0] == '*' {
		obj := TypeOfBFS(pkg, path, name[1:])
		if obj == nil {
			return nil
		}
		return types.NewPointer(obj)
	}

	obj := ObjectOfBFS(pkg, path, name)
	if obj == nil {
		return nil
	}

	return obj.Type()
}

func ObjectOfBFS(pkg *types.Package, path, name string) types.Object {
	lookupper := newLookupperBFS(pkg)
	return lookupper.Lookup(path, name)
}

type lookupperBFS struct {
	seen  map[*types.Package]struct{}
	queue *list.List
}

func newLookupperBFS(pkg *types.Package) *lookupperBFS {
	lookupper := &lookupperBFS{
		seen:  make(map[*types.Package]struct{}),
		queue: list.New(),
	}
	lookupper.queue.PushBack(pkg)
	return lookupper
}

func (lookup *lookupperBFS) Lookup(path, name string) types.Object {
	if lookup.queue.Len() == 0 {
		return nil
	}
	front := lookup.queue.Front()
	if front == nil {
		return nil
	}
	lookup.queue.Remove(front)
	pkg, ok := front.Value.(*types.Package)
	if !ok {
		return nil
	}
	if analysisutil.RemoveVendor(pkg.Path()) == analysisutil.RemoveVendor(path) {
		return pkg.Scope().Lookup(name)
	}
	for _, imp := range pkg.Imports() {
		if _, ok := lookup.seen[imp]; ok {
			continue
		}
		lookup.seen[imp] = struct{}{}
		lookup.queue.PushBack(imp)
	}
	return lookup.Lookup(path, name)
}

// copied and modified from https://github.com/gostaticanalysis/analysisutil/blob/ccfdecf515f47e636ba164ce0e5f26810eaf8747/types.go#L31
// TypeOf returns types.Type by given name in the package.
// TypeOf accepts pointer types such as *T.
func TypeOf(pass *analysis.Pass, pkg, name string) types.Type {
	if name == "" {
		return nil
	}

	if name[0] == '*' {
		obj := TypeOf(pass, pkg, name[1:])
		if obj == nil {
			return nil
		}
		return types.NewPointer(obj)
	}

	obj := ObjectOf(pass, pkg, name)
	if obj == nil {
		return nil
	}

	return obj.Type()
}

func MethodOf(typ types.Type, name string) *types.Func {
	return analysisutil.MethodOf(typ, name)
}
