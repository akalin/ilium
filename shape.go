package main

type Shape interface {
	// Returns whether or not the given ray intersects this
	// shape. If the ray intersects the shape and the given
	// intersection is not nil, also fills in that intersection.
	Intersect(ray *Ray, intersection *Intersection) bool
}
