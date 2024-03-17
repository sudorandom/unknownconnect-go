package unknownconnect

type option func(opts *interceptorOpts)

func WithDrop() option {
	return func(opts *interceptorOpts) {
		opts.drop = true
	}
}

func WithCallback(callback UnknownCallback) option {
	return func(opts *interceptorOpts) {
		opts.callbacks = append(opts.callbacks, callback)
	}
}
