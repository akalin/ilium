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
	return false
}
