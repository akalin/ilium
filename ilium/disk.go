package ilium

import "math"

const _DISK_EPSILON_SCALE float32 = 5e-4

type Disk struct {
	center  Point3
	i, j, k R3
	radius  float32
}

func MakeDisk(config map[string]interface{}) *Disk {
	centerConfig := config["center"].([]interface{})
	center := MakePoint3FromConfig(centerConfig)
	normalConfig := config["normal"].([]interface{})
	normal := MakeNormal3FromConfig(normalConfig)
	normal.Normalize(&normal)
	k := R3(normal)
	var i, j R3
	MakeCoordinateSystemNoAlias(&k, &i, &j)
	radius := float32(config["radius"].(float64))
	return &Disk{center, i, j, k, radius}
}

func (d *Disk) GetCenter() Point3 {
	return d.center
}

func (d *Disk) GetNormal() Normal3 {
	return Normal3(d.k)
}

func (d *Disk) Intersect(ray *Ray, intersection *Intersection) bool {
	dZ := ((*R3)(&ray.D)).Dot(&d.k)
	if absFloat32(dZ) < 1e-7 {
		return false
	}
	h := ((*R3)(&d.center)).Dot(&d.k)
	oZ := ((*R3)(&ray.O)).Dot(&d.k)
	tHit := (h - oZ) / dZ
	if tHit < ray.MinT || tHit > ray.MaxT {
		return false
	}
	pHit := ray.Evaluate(tHit)
	var r Vector3
	r.GetOffset(&d.center, &pHit)
	if r.NormSq() > d.radius*d.radius {
		return false
	}

	if intersection != nil {
		intersection.T = tHit
		intersection.P = pHit
		intersection.PEpsilon = _DISK_EPSILON_SCALE * intersection.T
		intersection.N = Normal3(d.k)
	}

	return true
}

func (d *Disk) SurfaceArea() float32 {
	return math.Pi * d.radius * d.radius
}

func (d *Disk) SampleSurface(u1, u2 float32) (
	pSurface Point3, pSurfaceEpsilon float32,
	nSurface Normal3, pdfSurfaceArea float32) {
	x, y := uniformSampleDisk(u1, u2)
	r := R3{x * d.radius, y * d.radius, 0}
	var rW R3
	rW.ConvertToCoordinateSystemNoAlias(&r, &d.i, &d.j, &d.k)
	v := Vector3(rW)
	pSurface.Shift(&d.center, &v)
	pSurfaceEpsilon = _DISK_EPSILON_SCALE * d.radius
	nSurface = Normal3(d.k)
	pdfSurfaceArea = 1 / d.SurfaceArea()
	return
}

func (d *Disk) SampleSurfaceFromPoint(
	u1, u2 float32, p Point3, pEpsilon float32, n Normal3) (
	pSurface Point3, pSurfaceEpsilon float32,
	nSurface Normal3, pdfSolidAngle float32) {
	return SampleEntireSurfaceFromPoint(d, u1, u2, p, pEpsilon, n)
}

func (d *Disk) ComputePdfFromPoint(
	p Point3, pEpsilon float32, n Normal3, wi Vector3) float32 {
	return ComputeEntireSurfacePdfFromPoint(d, p, pEpsilon, n, wi)
}
