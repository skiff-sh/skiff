package schema

import pluginv1alpha1 "github.com/skiff-sh/api/go/skiff/plugin/v1alpha1"

type DataSource interface {
	AddPackageEntries(packageName string, v ...Entry)
	Package(name string) PackageDataSource
	HasPackageEntry(packageName string, v Entry) bool
}

func NewDataSource() DataSource {
	return &dataSource{
		Map: make(map[string]PackageDataSource),
	}
}

type dataSource struct {
	Map map[string]PackageDataSource
}

func (d *dataSource) HasPackageEntry(packageName string, e Entry) bool {
	v := d.Map[packageName]
	if v == nil {
		return false
	}

	return v.Data()[e.FieldName()] != nil
}

func (d *dataSource) Package(name string) PackageDataSource {
	return d.Map[name]
}

func (d *dataSource) AddPackageEntries(packageName string, entries ...Entry) {
	for i := range entries {
		v := entries[i]
		pkg := d.Map[packageName]
		if pkg != nil {
			pkg.AddEntry(v)
		} else {
			d.Map[packageName] = NewPackageSource(v)
		}
	}
}

type EntryAdder interface {
	AddEntry(v Entry)
}

type PackageDataSource interface {
	EntryAdder
	Data() map[string]Value
	RawData() map[string]any
	PluginData() map[string]*pluginv1alpha1.Value
}

func NewPackageSource(vals ...Entry) PackageDataSource {
	return &packageDataSource{
		Sources: vals,
	}
}

type Entry interface {
	ValueSource
	// FieldName returns the name of the field.
	FieldName() string
}

type packageDataSource struct {
	PackageName string
	Sources     []Entry
}

func (d *packageDataSource) PluginData() map[string]*pluginv1alpha1.Value {
	out := map[string]*pluginv1alpha1.Value{}

	for _, v := range d.Sources {
		out[v.FieldName()] = v.Value().Plugin()
	}

	return out
}

func (d *packageDataSource) RawData() map[string]any {
	out := map[string]any{}

	for _, v := range d.Sources {
		out[v.FieldName()] = v.Value().Any()
	}

	return out
}

func (d *packageDataSource) Data() map[string]Value {
	out := map[string]Value{}

	for _, v := range d.Sources {
		out[v.FieldName()] = v.Value()
	}

	return out
}

func (d *packageDataSource) Package() string {
	return d.PackageName
}

func (d *packageDataSource) AddEntry(v Entry) {
	d.Sources = append(d.Sources, v)
}
