package fetcher

import "fmt"

type Registry struct {
	sources map[string]Fetcher
}

func NewRegistry() *Registry {
	return &Registry{sources: make(map[string]Fetcher)}
}

func (r *Registry) Register(f Fetcher) error {
	if f == nil {
		return fmt.Errorf("nil fetcher")
	}
	name := f.Name()
	if name == "" {
		return fmt.Errorf("fetcher has empty name")
	}
	if _, exists := r.sources[name]; exists {
		return fmt.Errorf("fetcher %q already registered", name)
	}
	r.sources[name] = f
	return nil
}

func (r *Registry) Get(name string) (Fetcher, bool) {
	f, ok := r.sources[name]
	return f, ok
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.sources))
	for n := range r.sources {
		names = append(names, n)
	}
	return names
}
