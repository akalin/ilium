package ilium

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

func (tr *Triangle) Intersect(ray *Ray, intersection *Intersection) bool {
	p1 := &tr.mesh.vertices[tr.mesh.indices[tr.i][0]]
	p2 := &tr.mesh.vertices[tr.mesh.indices[tr.i][1]]
	p3 := &tr.mesh.vertices[tr.mesh.indices[tr.i][2]]
	var e1, e2 Vector3
	e1.GetOffset(p1, p2)
	e2.GetOffset(p1, p3)
	var s1 Vector3
	s1.CrossNoAlias(&ray.D, &e2)
	divisor := s1.Dot(&e1)
	if divisor == 0.0 {
		return false
	}
	invDivisor := 1.0 / divisor

	var d Vector3
	d.GetOffset(p1, &ray.O)
	b1 := d.Dot(&s1) * invDivisor
	if b1 < 0.0 || b1 > 1.0 {
		return false
	}

	var s2 Vector3
	s2.CrossNoAlias(&d, &e1)
	b2 := ray.D.Dot(&s2) * invDivisor
	if b2 < 0.0 || (b1+b2) > 1.0 {
		return false
	}

	t := e2.Dot(&s2) * invDivisor
	if t < ray.MinT || t > ray.MaxT {
		return false
	}

	intersection.T = t
	intersection.P = ray.Evaluate(intersection.T)
	intersection.PEpsilon = 1e-3 * intersection.T
	intersection.N.CrossVectorNoAlias(&e1, &e2)
	intersection.N.Normalize(&intersection.N)
	return true
}

func (tr *Triangle) GetBoundingBox() BBox {
	p1 := &tr.mesh.vertices[tr.mesh.indices[tr.i][0]]
	p2 := &tr.mesh.vertices[tr.mesh.indices[tr.i][1]]
	p3 := &tr.mesh.vertices[tr.mesh.indices[tr.i][2]]
	boundingBox := MakeInvalidBBox()
	boundingBox = boundingBox.UnionPoint(*p1)
	boundingBox = boundingBox.UnionPoint(*p2)
	boundingBox = boundingBox.UnionPoint(*p3)
	return boundingBox
}

func (t *Triangle) MayIntersectBoundingBox(boundingBox BBox) bool {
	return true
}
