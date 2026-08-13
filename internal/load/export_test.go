package load

import (
	"context"
	"net"
)

// SetTestDial replaces the network dial used by Restricted pinned-IP
// connections. Tests use this to assert the loader dials ip:port literals
// and never re-dials the hostname.
func (l *Loader) SetTestDial(fn func(ctx context.Context, network, address string) (net.Conn, error)) {
	l.testDial = fn
}
