package main

type Shape interface {
	Intersect(ray *Ray, intersection *Intersection) bool
}
