package main

type GridAggregate struct {
	boundingBox BBox
	nVoxels     [3]int
	voxels      []PrimitiveList
}

func (g *GridAggregate) Intersect(ray *Ray, intersection *Intersection) bool {
	return g.voxels[0].Intersect(ray, intersection)
}

func (g *GridAggregate) GetBoundingBox() BBox {
	return g.boundingBox
}

func (g *GridAggregate) GetSensors() []Sensor {
	sensors := []Sensor{}
	for _, voxel := range g.voxels {
		sensors = append(sensors, voxel.GetSensors()...)
	}
	return sensors
}

func MakeGridAggregate(
	config map[string]interface{}, primitives []Primitive) Aggregate {
	allPrimitives := PrimitiveList{primitives}
	boundingBox := allPrimitives.GetBoundingBox()
	nVoxels := [3]int{1, 1, 1}
	voxels := []PrimitiveList{allPrimitives}
	return &GridAggregate{boundingBox, nVoxels, voxels}
}
