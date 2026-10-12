package errors

// Wrap augments any error into an internal error preserving the chain.
func Wrap(err error) error {
	if err == nil {
		return nil
	}
	return New(err.Error(), "internal", 13 /* Internal */, 500).Wrap(err)
}

// Message returns the root message of the error chain.
func Message(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}