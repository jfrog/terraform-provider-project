package project

type Set[T Equatable] []T

func SetFromSlice[T Equatable](values []T) Set[T] {
	set := make(Set[T], 0)
	set = append(set, values...)
	return set
}

func (s Set[T]) Contains(b T) bool {
	for _, a := range s {
		if a.Equals(b) {
			return true
		}
	}
	return false
}

// Intersection returns a Set containing all the common items between both Sets.
// Example: [1, 2, 3].Intersection([2, 3, 4]) = [2, 3].
func (s Set[T]) Intersection(other Set[T]) Set[T] {
	intersection := make(Set[T], 0)
	for _, item := range s {
		if other.Contains(item) {
			intersection = append(intersection, item)
		}
	}
	return intersection
}

// Difference returns a Set containing all the items not contained in the other set.
// Note this is "unidirectional", and the result is _only_ the elements in A that are not in B.
// Example: [1, 2, 3].Difference([2, 3, 4]) = [1].
func (s Set[T]) Difference(other Set[T]) Set[T] {
	diff := make(Set[T], 0)
	for _, item := range s {
		if !other.Contains(item) {
			diff = append(diff, item)
		}
	}
	return diff
}
