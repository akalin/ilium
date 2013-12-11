package ilium

import "fmt"
import "math/rand"

type PathContext struct {
	RussianRouletteState *RussianRouletteState
	LightBundle          SampleBundle
	SensorBundle         SampleBundle
	ChooseLightSample    Sample1D
	LightWiSamples       Sample2DArray
	SensorWiSamples      Sample2DArray
	Scene                *Scene
	Sensor               Sensor
	X, Y                 int
}

type pathVertexType int

const (
	_PATH_VERTEX_LIGHT_SUPER_VERTEX         pathVertexType = iota
	_PATH_VERTEX_SENSOR_SUPER_VERTEX        pathVertexType = iota
	_PATH_VERTEX_LIGHT_VERTEX               pathVertexType = iota
	_PATH_VERTEX_SENSOR_VERTEX              pathVertexType = iota
	_PATH_VERTEX_SURFACE_INTERACTION_VERTEX pathVertexType = iota
)

type PathVertex struct {
	vertexType    pathVertexType
	transportType MaterialTransportType
	p             Point3
	pEpsilon      float32
	n             Normal3
	alpha         Spectrum
	// Used by light and surface interaction vertices only.
	light Light
	// Used by surface interaction vertices only.
	sensor   Sensor
	material Material
}

func MakeLightSuperVertex() PathVertex {
	return PathVertex{
		vertexType:    _PATH_VERTEX_LIGHT_SUPER_VERTEX,
		transportType: MATERIAL_IMPORTANCE_TRANSPORT,
		alpha:         MakeConstantSpectrum(1),
	}
}

func MakeSensorSuperVertex() PathVertex {
	return PathVertex{
		vertexType:    _PATH_VERTEX_SENSOR_SUPER_VERTEX,
		transportType: MATERIAL_LIGHT_TRANSPORT,
		alpha:         MakeConstantSpectrum(1),
	}
}

func (pv *PathVertex) initializeSurfaceInteractionVertex(
	context *PathContext, pvPrev *PathVertex, intersection *Intersection,
	alpha Spectrum) {
	var sensor Sensor
	for i := 0; i < len(intersection.Sensors); i++ {
		if intersection.Sensors[i] == context.Sensor {
			sensor = context.Sensor
			break
		}
	}
	*pv = PathVertex{
		vertexType:    _PATH_VERTEX_SURFACE_INTERACTION_VERTEX,
		transportType: pvPrev.transportType,
		p:             intersection.P,
		pEpsilon:      intersection.PEpsilon,
		n:             intersection.N,
		alpha:         alpha,
		light:         intersection.Light,
		sensor:        sensor,
		material:      intersection.Material,
	}
}

func validateSampledPathEdge(context *PathContext, pv, pvNext *PathVertex) {
	if pv != nil && pv.transportType != pvNext.transportType {
		panic(fmt.Sprintf("Sampled path edge with non-matching "+
			"transport types %v -> %v", pv, pvNext))
	}

	switch {
	case pv == nil:
		if pvNext.vertexType == _PATH_VERTEX_LIGHT_SUPER_VERTEX {
			return
		}

		if pvNext.vertexType == _PATH_VERTEX_SENSOR_SUPER_VERTEX {
			return
		}

	case pv.vertexType == _PATH_VERTEX_LIGHT_SUPER_VERTEX:
		if pvNext.vertexType == _PATH_VERTEX_LIGHT_VERTEX {
			return
		}

	case pv.vertexType == _PATH_VERTEX_SENSOR_SUPER_VERTEX:
		if pvNext.vertexType == _PATH_VERTEX_SENSOR_VERTEX {
			return
		}

	case pv.vertexType == _PATH_VERTEX_LIGHT_VERTEX:
		if pvNext.vertexType ==
			_PATH_VERTEX_SURFACE_INTERACTION_VERTEX {
			return
		}

	case pv.vertexType == _PATH_VERTEX_SENSOR_VERTEX:
		if pvNext.vertexType ==
			_PATH_VERTEX_SURFACE_INTERACTION_VERTEX {
			return
		}

	case pv.vertexType == _PATH_VERTEX_SURFACE_INTERACTION_VERTEX:
		if pvNext.vertexType ==
			_PATH_VERTEX_SURFACE_INTERACTION_VERTEX {
			return
		}
	}

	panic(fmt.Sprintf("Invalid sampled path edge %v -> %v", pv, pvNext))
}

func (pv *PathVertex) shouldContinue(
	context *PathContext, sampleIndex int, albedo *Spectrum,
	rng *rand.Rand) bool {
	pContinue := context.RussianRouletteState.GetContinueProbability(
		sampleIndex, albedo)
	if pContinue <= 0 {
		return false
	}
	if pContinue < 1 {
		if randFloat32(rng) > pContinue {
			return false
		}
		albedo.ScaleInv(albedo, pContinue)
	}
	return true
}

func (pv *PathVertex) SampleNext(
	context *PathContext, i int, rng *rand.Rand,
	pvPrev, pvNext *PathVertex) bool {
	validateSampledPathEdge(context, pvPrev, pv)

	switch pv.vertexType {
	case _PATH_VERTEX_LIGHT_SUPER_VERTEX:
		light, _ :=
			context.Scene.SampleLight(context.ChooseLightSample.U)
		p, pEpsilon, n, LeSpatialDivPdf, pdfSpatial :=
			light.SampleSurface(context.LightBundle)
		if LeSpatialDivPdf.IsBlack() || pdfSpatial == 0 {
			return false
		}

		albedo := &LeSpatialDivPdf
		if !pv.shouldContinue(context, i, albedo, rng) {
			return false
		}

		var alphaNext Spectrum
		alphaNext.Mul(&pv.alpha, albedo)
		*pvNext = PathVertex{
			vertexType:    _PATH_VERTEX_LIGHT_VERTEX,
			transportType: pv.transportType,
			p:             p,
			pEpsilon:      pEpsilon,
			n:             n,
			alpha:         alphaNext,
			light:         light,
		}

	case _PATH_VERTEX_SENSOR_SUPER_VERTEX:
		p, pEpsilon, n, WeSpatialDivPdf, pdfSpatial :=
			context.Sensor.SampleSurface(context.SensorBundle)
		if WeSpatialDivPdf.IsBlack() || pdfSpatial == 0 {
			return false
		}

		albedo := &WeSpatialDivPdf
		if !pv.shouldContinue(context, i, albedo, rng) {
			return false
		}

		var alphaNext Spectrum
		alphaNext.Mul(&pv.alpha, albedo)
		*pvNext = PathVertex{
			vertexType:    _PATH_VERTEX_SENSOR_VERTEX,
			transportType: pv.transportType,
			p:             p,
			pEpsilon:      pEpsilon,
			n:             n,
			alpha:         alphaNext,
		}

	case _PATH_VERTEX_LIGHT_VERTEX:
		wo, LeDirectionalDivPdf, pdfDirectional :=
			pv.light.SampleDirection(
				context.LightBundle, pv.p, pv.n)
		if LeDirectionalDivPdf.IsBlack() || pdfDirectional == 0 {
			return false
		}

		albedo := &LeDirectionalDivPdf
		if !pv.shouldContinue(context, i, albedo, rng) {
			return false
		}

		ray := Ray{pv.p, wo, pv.pEpsilon, infFloat32(+1)}
		var intersection Intersection
		if !context.Scene.Aggregate.Intersect(&ray, &intersection) {
			return false
		}

		var alphaNext Spectrum
		alphaNext.Mul(&pv.alpha, albedo)
		pvNext.initializeSurfaceInteractionVertex(
			context, pv, &intersection, alphaNext)

	case _PATH_VERTEX_SENSOR_VERTEX:
		wo, WeDirectionalDivPdf, pdfDirectional :=
			context.Sensor.SampleDirection(
				context.X, context.Y, context.SensorBundle,
				pv.p, pv.n)
		if WeDirectionalDivPdf.IsBlack() || pdfDirectional == 0 {
			return false
		}

		albedo := &WeDirectionalDivPdf
		if !pv.shouldContinue(context, i, albedo, rng) {
			return false
		}

		ray := Ray{pv.p, wo, pv.pEpsilon, infFloat32(+1)}
		var intersection Intersection
		if !context.Scene.Aggregate.Intersect(&ray, &intersection) {
			return false
		}

		var alphaNext Spectrum
		alphaNext.Mul(&pv.alpha, albedo)
		pvNext.initializeSurfaceInteractionVertex(
			context, pv, &intersection, alphaNext)

	case _PATH_VERTEX_SURFACE_INTERACTION_VERTEX:
		var wo Vector3
		wo.GetDirectionAndDistance(&pv.p, &pvPrev.p)

		var wiSamples Sample2DArray
		switch pv.transportType {
		case MATERIAL_LIGHT_TRANSPORT:
			wiSamples = context.SensorWiSamples
		case MATERIAL_IMPORTANCE_TRANSPORT:
			wiSamples = context.LightWiSamples
		}
		// Subtract one for the super-vertex, and one for the
		// light/sensor vertex.
		sampleIndex := i - 2
		sample := wiSamples.GetSample(sampleIndex, rng)
		wi, fAbsDivPdf, pdf := pv.material.SampleWi(
			pv.transportType, sample.U1, sample.U2, wo, pv.n)
		if fAbsDivPdf.IsBlack() || pdf == 0 {
			return false
		}

		albedo := &fAbsDivPdf
		if !pv.shouldContinue(context, i, albedo, rng) {
			return false
		}

		ray := Ray{pv.p, wi, pv.pEpsilon, infFloat32(+1)}
		var intersection Intersection
		if !context.Scene.Aggregate.Intersect(&ray, &intersection) {
			return false
		}

		var alphaNext Spectrum
		alphaNext.Mul(&pv.alpha, albedo)
		pvNext.initializeSurfaceInteractionVertex(
			context, pv, &intersection, alphaNext)

	default:
		panic(fmt.Sprintf(
			"Unknown path vertex type %d", pv.vertexType))
	}

	validateSampledPathEdge(context, pv, pvNext)
	return true
}

func validateConnectingPathEdge(context *PathContext, pv, pvOther *PathVertex) {
	// TODO(akalin): Implement.
}

func (pv *PathVertex) ComputeUnweightedContribution(
	context *PathContext,
	pvPrev, pvOther, pvOtherPrev *PathVertex) Spectrum {
	validateSampledPathEdge(context, pvPrev, pv)
	validateSampledPathEdge(context, pvOtherPrev, pvOther)
	validateConnectingPathEdge(context, pv, pvOther)

	switch pv.vertexType {
	case _PATH_VERTEX_LIGHT_SUPER_VERTEX:
		return Spectrum{}

	case _PATH_VERTEX_SENSOR_SUPER_VERTEX:
		return Spectrum{}

	case _PATH_VERTEX_LIGHT_VERTEX:
		return Spectrum{}

	case _PATH_VERTEX_SENSOR_VERTEX:
		return Spectrum{}

	case _PATH_VERTEX_SURFACE_INTERACTION_VERTEX:
		return Spectrum{}
	}
	return Spectrum{}
}