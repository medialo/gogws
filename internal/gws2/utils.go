package gws2

func ptr[T any](v T) *T {
	return &v
}
