package grpc

import (
"google.golang.org/grpc/naming"
)

type staticResolver struct{
	updates []*naming.Update
}

type staticWatcher struct {
	updates chan []*naming.Update
}

func NewStaticResolver(addr []string) naming.Resolver {
	var ups []*naming.Update
	for _,a := range addr {
		ups = append(ups, &naming.Update{naming.Add, a, ""})
	}
	return &staticResolver{ups}
}

func (w *staticWatcher) Next() ([]*naming.Update, error) {
	return <-w.updates, nil
}

func (w *staticWatcher) Close() {
	close(w.updates)
}

func (r *staticResolver) Resolve(target string) (naming.Watcher, error) {
	var ch chan []*naming.Update = make(chan []*naming.Update, 1)
	ch <- r.updates
	return &staticWatcher{ch}, nil
}
