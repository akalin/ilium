package ilium

const _TRIANGLE_EPSILON_SCALE float32 = 1e-3

type triangleMesh struct {
	vertices []Point3
	indices  [][3]int
}

type Triangle struct {
	mesh *triangleMesh
	i    int
}

func makeIndicesFromConfig(config interface{}) [][3]int {
	arrayConfig := config.([]interface{})
	indices := [][3]int{}
	for i := 0; i < len(arrayConfig); i += 3 {
		indices = append(
			indices,
			[3]int{
				int(arrayConfig[i+0].(float64)),
				int(arrayConfig[i+1].(float64)),
				int(arrayConfig[i+2].(float64)),
			})
	}
	return indices
}

func MakeTriangleMesh(config map[string]interface{}) []Shape {
	verticesConfig := config["vertices"].([]interface{})
	vertices := MakePoint3sFromConfig(verticesConfig)
	indicesConfig := config["indices"].([]interface{})
	indices := makeIndicesFromConfig(indicesConfig)
	mesh := triangleMesh{vertices, indices}
	triangles := make([]Shape, len(indices))
	for i := 0; i < len(triangles); i++ {
		triangles[i] = &Triangle{&mesh, i}
	}
	return triangles
}

func (tr *Triangle) getVertices() (p1, p2, p3 *Point3) {
	p1 = &tr.mesh.vertices[tr.mesh.indices[tr.i][0]]
	p2 = &tr.mesh.vertices[tr.mesh.indices[tr.i][1]]
	p3 = &tr.mesh.vertices[tr.mesh.indices[tr.i][2]]
	return
}

func (tr *Triangle) getEVectors() (e1, e2 Vector3, n Normal3) {
	p1, p2, p3 := tr.getVertices()
	e1.GetOffset(p1, p2)
	e2.GetOffset(p1, p3)
	n.CrossVectorNoAlias(&e1, &e2)
	return
}

func (tr *Triangle) Intersect(ray *Ray, intersection *Intersection) bool {
	p1, _, _ := tr.getVertices()
	e1, e2, n := tr.getEVectors()
	var s1 Vector3
	s1.CrossNoAlias(&ray.D, &e2)
	divisor := s1.Dot(&e1)
	if divisor == 0 {
		return false
	}
	invDivisor := 1 / divisor

	var d Vector3
	d.GetOffset(p1, &ray.O)
	b1 := d.Dot(&s1) * invDivisor
	if b1 < 0 || b1 > 1 {
		return false
	}

	var s2 Vector3
	s2.CrossNoAlias(&d, &e1)
	b2 := ray.D.Dot(&s2) * invDivisor
	if b2 < 0 || (b1+b2) > 1 {
		return false
	}

	t := e2.Dot(&s2) * invDivisor
	if t < ray.MinT || t > ray.MaxT {
		return false
	}

	if intersection != nil {
		intersection.T = t
		intersection.P = ray.Evaluate(intersection.T)
		intersection.PEpsilon = _TRIANGLE_EPSILON_SCALE * intersection.T
		intersection.N.CrossVectorNoAlias(&e1, &e2)
		intersection.N.Normalize(&intersection.N)
		intersection.N.Normalize(&n)
	}

	return true
}

func (tr *Triangle) SurfaceArea() float32 {
	_, _, n := tr.getEVectors()
	return 0.5 * n.Norm()
}

func (tr *Triangle) SampleSurface(u1, u2 float32) (
	pSurface Point3, pSurfaceEpsilon float32,
	nSurface Normal3, pdfSurfaceArea float32) {
	b1, b2 := uniformSampleTriangle(u1, u2)
	p1, p2, p3 := tr.getVertices()
	var r1, r2, r3 R3
	r1.Scale((*R3)(p1), b1)
	r2.Scale((*R3)(p2), b2)
	r3.Scale((*R3)(p3), 1-b1-b2)
	var rSurface R3
	rSurface.Add(&r1, &r2)
	rSurface.Add(&rSurface, &r3)
	pSurface = Point3(rSurface)
	pSurfaceEpsilon =
		_TRIANGLE_EPSILON_SCALE * sqrtFloat32(tr.SurfaceArea())
	_, _, n := tr.getEVectors()
	nSurface.Normalize(&n)
	pdfSurfaceArea = 1 / tr.SurfaceArea()
	return
}

func (tr *Triangle) SampleSurfaceFromPoint(
	u1, u2 float32, p Point3, n Normal3) (
	pSurface Point3, pSurfaceEpsilon float32, nSurface Normal3,
	pdfProjectedSolidAngle float32) {
	return SampleEntireSurfaceFromPoint(tr, u1, u2, p, n)
}
