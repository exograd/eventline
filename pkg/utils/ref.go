package utils

func Ref[T any](x T) *T {
	return &x
}
