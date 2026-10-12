package events

type event[T any] interface {
	Type() T
}
