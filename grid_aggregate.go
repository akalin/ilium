package main

type GridAggregate struct {
	boundingBox BBox
	nVoxels     [3]int
	widths      [3]float32
	voxels      []PrimitiveList
}

func argMin3(x [3]float32) int {
	if x[0] < x[1] {
		if x[0] < x[2] {
			return 0
		}
		return 2
	}

	if x[1] < x[2] {
		return 1
	}

	return 2
}

func (g *GridAggregate) getVoxel(voxelPos [3]int) PrimitiveList {
	return g.voxels[0]
}

func (g *GridAggregate) pointToVoxelPosI(
	pointI float32, i int) (voxelPosI int) {
	return 0
}

func (g *GridAggregate) voxelPosToPointI(
	voxelPosI, i int) (pointI float32) {
	PMin := ((*R3)(&g.boundingBox.PMin)).ToArray()
	return PMin[i]
}

func (g *GridAggregate) Intersect(ray *Ray, intersection *Intersection) bool {
	var tHitMin float32
	if g.boundingBox.Contains(ray.Evaluate(ray.MinT)) {
		tHitMin = ray.MinT
	} else {
		var tHitMax float32
		if !g.boundingBox.IntersectRay(ray, &tHitMin, &tHitMax) {
			return false
		}
	}

	rayMin := R3(ray.Evaluate(tHitMin))

	p := rayMin.ToArray()
	d := ((*R3)(&ray.D)).ToArray()
	// Both nextCrossingT and deltaT can contain infinities, but
	// everything should still work regardless.
	var nextCrossingT, deltaT [3]float32
	var vp, step, out [3]int
	for i := 0; i < 3; i++ {
		vp[i] = g.pointToVoxelPosI(p[i], i)
		if d[i] >= 0 {
			dpI := g.voxelPosToPointI(vp[i]+1, i) - p[i]
			nextCrossingT[i] = tHitMin + dpI/d[i]
			deltaT[i] = g.widths[i] / d[i]
			step[i] = 1
			out[i] = g.nVoxels[i]
		} else {
			dpI := g.voxelPosToPointI(vp[i], i) - p[i]
			nextCrossingT[i] = tHitMin + dpI/d[i]
			deltaT[i] = -g.widths[i] / d[i]
			step[i] = -1
			out[i] = -1
		}
	}

	found := false
	tempRay := *ray
	for {
		v := g.getVoxel(vp)
		if v.Intersect(&tempRay, intersection) {
			tempRay.MaxT = intersection.T
			found = true
		}

		minI := argMin3(nextCrossingT)

		if nextCrossingT[minI] > tempRay.MaxT {
			break
		}
		vp[minI] += step[minI]
		if vp[minI] == out[minI] {
			break
		}
		nextCrossingT[minI] += deltaT[minI]
	}

	return found
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
	var widthV Vector3
	widthV.GetOffset(&boundingBox.PMin, &boundingBox.PMax)
	widths := ((*R3)(&widthV)).ToArray()
	voxels := []PrimitiveList{allPrimitives}
	return &GridAggregate{boundingBox, nVoxels, widths, voxels}
}
