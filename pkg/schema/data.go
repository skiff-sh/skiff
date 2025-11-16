package schema

type Data interface {
	AddPackageEntry(packageName string, v Entry)
	Package(name string) PackageData
	HasPackageEntry(packageName string, v Entry) bool
}

func NewData() Data {
	return &data{
		Map: make(map[string]PackageData),
	}
}

type data struct {
	Map map[string]PackageData
}

func (d *data) HasPackageEntry(packageName string, e Entry) bool {
	v := d.Map[packageName]
	if v == nil {
		return false
	}

	return v.Data()[e.FieldName()] != nil
}

func (d *data) Package(name string) PackageData {
	return d.Map[name]
}

func (d *data) AddPackageEntry(packageName string, v Entry) {
	pkg := d.Map[packageName]
	if pkg != nil {
		pkg.AddEntry(v)
	} else {
		d.Map[packageName] = NewPackageSource(v)
	}
}

type EntryAdder interface {
	AddEntry(v Entry)
}

type PackageData interface {
	EntryAdder
	Data() map[string]Value
	RawData() map[string]any
}

func NewPackageSource(vals ...Entry) PackageData {
	return &dataSource{
		Sources: vals,
	}
}

type Entry interface {
	ValueSource
	// FieldName returns the name of the field.
	FieldName() string
}

type dataSource struct {
	PackageName string
	Sources     []Entry
}

func (d *dataSource) RawData() map[string]any {
	out := map[string]any{}

	for _, v := range d.Sources {
		out[v.FieldName()] = v.Value().Any()
	}

	return out
}

func (d *dataSource) Data() map[string]Value {
	out := map[string]Value{}

	for _, v := range d.Sources {
		out[v.FieldName()] = v.Value()
	}

	return out
}

func (d *dataSource) Package() string {
	return d.PackageName
}

func (d *dataSource) AddEntry(v Entry) {
	d.Sources = append(d.Sources, v)
}
