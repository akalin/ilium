package main

type Point3 R3

func (out *Point3) Shift(p *Point3, offset *Vector3) {
	((*R3)(out)).Add((*R3)(p), (*R3)(offset))
}
