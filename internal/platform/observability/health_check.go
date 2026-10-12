package observability

// healthCheck is a liveness probe: it runs synchronously and reports the
// first error (if any) on each /healthz call.
type healthCheck struct {
	name  string
	check func() error
}

func (hc *healthCheck) do() error {
	if hc.check == nil {
		return nil
	}
	return hc.check()
}