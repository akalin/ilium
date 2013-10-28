package ilium

type Normal3 R3

func (out *Normal3) Flip(n *Normal3) {
	((*R3)(out)).Invert((*R3)(n))
}

func (out *Normal3) Normalize(n *Normal3) {
	((*R3)(out)).Normalize((*R3)(n))
}
