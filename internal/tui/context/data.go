package tuicontext

// Data is a small typed metadata bag for cross-component state.
type Data struct {
	values map[string]string
}

func NewData() Data {
	return Data{values: map[string]string{}}
}

func (d *Data) Set(key, value string) {
	if d.values == nil {
		d.values = map[string]string{}
	}
	d.values[key] = value
}

func (d Data) Get(key string) (string, bool) {
	v, ok := d.values[key]
	return v, ok
}
