package shop

type Page[T Priced] struct {
	Items []T
	Next  *string
}
