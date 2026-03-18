package metrics

import "net"

// newListener creates a TCP listener on addr.
// It is factored out of ServeMetrics so the net import stays separate from
// the Prometheus-heavy metrics.go file.
func newListener(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}
