package ilium

type voxel struct {
	primitives []Primitive
}

type GridAggregate struct {
	boundingBox       BBox
	nVoxels           [3]int
	widths, invWidths [3]float32
	voxels            []voxel
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

func max3(x [3]float32) float32 {
	return maxFloat32(maxFloat32(x[0], x[1]), x[2])
}

func (g *GridAggregate) getVoxel(voxelPos [3]int) *voxel {
	i := voxelPos[2]*g.nVoxels[0]*g.nVoxels[1] +
		voxelPos[1]*g.nVoxels[0] + voxelPos[0]
	return &g.voxels[i]
}

func (g *GridAggregate) pointToVoxelPosI(
	pointI float32, i int) (voxelPosI int) {
	PMin := ((*R3)(&g.boundingBox.PMin)).ToArray()
	voxelPosI = int((pointI - PMin[i]) * g.invWidths[i])
	if voxelPosI < 0 {
		voxelPosI = 0
	}
	if voxelPosI >= g.nVoxels[i] {
		voxelPosI = g.nVoxels[i] - 1
	}
	return
}

func (g *GridAggregate) voxelPosToPointI(
	voxelPosI, i int) (pointI float32) {
	PMin := ((*R3)(&g.boundingBox.PMin)).ToArray()
	return PMin[i] + float32(voxelPosI)*g.widths[i]
}

func (g *GridAggregate) getVoxelBoundingBox(voxelPos [3]int) BBox {
	PMin := Point3{
		g.voxelPosToPointI(voxelPos[0], 0),
		g.voxelPosToPointI(voxelPos[1], 1),
		g.voxelPosToPointI(voxelPos[2], 2),
	}
	PMax := Point3{
		g.voxelPosToPointI(voxelPos[0]+1, 0),
		g.voxelPosToPointI(voxelPos[1]+1, 1),
		g.voxelPosToPointI(voxelPos[2]+1, 2),
	}
	return BBox{PMin, PMax}
}

func (g *GridAggregate) addPrimitives(primitives []Primitive) {
	for _, primitive := range primitives {
		boundingBox := primitive.GetBoundingBox()
		PMin := ((*R3)(&boundingBox.PMin)).ToArray()
		PMax := ((*R3)(&boundingBox.PMax)).ToArray()
		var vMin, vMax [3]int
		for i := 0; i < 3; i++ {
			vMin[i] = g.pointToVoxelPosI(PMin[i], i)
			vMax[i] = g.pointToVoxelPosI(PMax[i], i)
		}

		for z := vMin[2]; z <= vMax[2]; z++ {
			for y := vMin[1]; y <= vMax[1]; y++ {
				for x := vMin[0]; x <= vMax[0]; x++ {
					voxelPos := [3]int{x, y, z}
					vBoundingBox :=
						g.getVoxelBoundingBox(voxelPos)
					if primitive.MayIntersectBoundingBox(
						vBoundingBox) {
						v := g.getVoxel(voxelPos)
						v.primitives =
							append(
								v.primitives,
								primitive)
					}
				}
			}
		}
	}
}

func (g *GridAggregate) MayIntersectBoundingBox(boundingBox BBox) bool {
	return true
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
		p := PrimitiveList{g.getVoxel(vp).primitives}
		if p.Intersect(&tempRay, intersection) {
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
		p := PrimitiveList{voxel.primitives}
		sensors = append(sensors, p.GetSensors()...)
	}
	return sensors
}

func MakeGridAggregate(
	config map[string]interface{}, primitives []Primitive) Aggregate {
	allPrimitives := PrimitiveList{primitives}
	boundingBox := allPrimitives.GetBoundingBox()
	var widthV Vector3
	widthV.GetOffset(&boundingBox.PMin, &boundingBox.PMax)
	boundingWidths := ((*R3)(&widthV)).ToArray()
	voxelsPerUnitDistance :=
		3 * powFloat32(float32(len(primitives)), 1/3) /
			max3(boundingWidths)

	var nVoxels [3]int
	var widths, invWidths [3]float32
	for i := 0; i < 3; i++ {
		nVoxels[i] =
			int(boundingWidths[i]*voxelsPerUnitDistance + 0.5)
		if nVoxels[i] < 1 {
			nVoxels[i] = 1
		} else if nVoxels[i] > 64 {
			nVoxels[i] = 64
		}
		widths[i] = boundingWidths[i] / float32(nVoxels[i])
		if widths[i] != 0 {
			invWidths[i] = 1 / widths[i]
		}
	}

	totalVoxels := nVoxels[0] * nVoxels[1] * nVoxels[2]
	voxels := make([]voxel, totalVoxels)
	grid := &GridAggregate{boundingBox, nVoxels, widths, invWidths, voxels}
	grid.addPrimitives(primitives)
	return grid
}
