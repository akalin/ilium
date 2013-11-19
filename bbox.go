package main

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
