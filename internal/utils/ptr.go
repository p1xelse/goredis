package utils

func Ptr[T any](v T) *T {
	return &v
}

func PtrOrNil[T comparable](v T) *T {
	if z, ok := any(v).(interface{ IsZero() bool }); ok {
		if z.IsZero() {
			return nil
		}
		return &v
	}

	var zero T
	if v == zero {
		return nil
	}
	return &v
}

func Deref[T any](p *T) T {
	if p != nil {
		return *p
	}
	return *new(T)
}
