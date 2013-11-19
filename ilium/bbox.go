package ilium

type BBox struct {
	PMin, PMax Point3
}

func MakeInvalidBBox() BBox {
	positiveInf := infFloat32(+1)
	negativeInf := infFloat32(-1)
	PMin := Point3{positiveInf, positiveInf, positiveInf}
	PMax := Point3{negativeInf, negativeInf, negativeInf}
	return BBox{PMin, PMax}
}

func (b *BBox) Contains(p Point3) bool {
	return p.X >= b.PMin.X && p.X <= b.PMax.X &&
		p.Y >= b.PMin.Y && p.Y <= b.PMax.Y &&
		p.Z >= b.PMin.Z && p.Z <= b.PMax.Z
}

func (b *BBox) IntersectRay(r *Ray, tHitMin, tHitMax *float32) bool {
	t0 := r.MinT
	t1 := r.MaxT
	PMin := ((*R3)(&b.PMin)).ToArray()
	PMax := ((*R3)(&b.PMax)).ToArray()
	o := ((*R3)(&r.O)).ToArray()
	d := ((*R3)(&r.D)).ToArray()

	for i := 0; i < 3; i++ {
		// invD can end up being +/-infinity but everything
		// should still work regardless.
		invD := 1 / d[i]
		tNear := (PMin[i] - o[i]) * invD
		tFar := (PMax[i] - o[i]) * invD
		if tNear > tFar {
			tNear, tFar = tFar, tNear
		}
		t0 = maxFloat32(t0, tNear)
		t1 = minFloat32(t1, tFar)
		if t0 > t1 {
			return false
		}
	}

	*tHitMin = t0
	*tHitMax = t1
	return true
}

func (b BBox) Union(c BBox) BBox {
	PMin := Point3{
		minFloat32(b.PMin.X, c.PMin.X),
		minFloat32(b.PMin.Y, c.PMin.Y),
		minFloat32(b.PMin.Z, c.PMin.Z),
	}
	PMax := Point3{
		maxFloat32(b.PMax.X, c.PMax.X),
		maxFloat32(b.PMax.Y, c.PMax.Y),
		maxFloat32(b.PMax.Z, c.PMax.Z),
	}
	return BBox{PMin, PMax}
}
