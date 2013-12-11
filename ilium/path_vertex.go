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
	_PATH_VERTEX_LIGHT_SUPER_VERTEX  pathVertexType = iota
	_PATH_VERTEX_SENSOR_SUPER_VERTEX pathVertexType = iota
	_PATH_VERTEX_LIGHT_VERTEX        pathVertexType = iota
	_PATH_VERTEX_SENSOR_VERTEX       pathVertexType = iota
)

type PathVertex struct {
	vertexType pathVertexType
	p          Point3
	pEpsilon   float32
	n          Normal3
	alpha      Spectrum
	// Used by light vertices only.
	light Light
}

func MakeLightSuperVertex() PathVertex {
	return PathVertex{
		vertexType: _PATH_VERTEX_LIGHT_SUPER_VERTEX,
		alpha:      MakeConstantSpectrum(1),
	}
}

func MakeSensorSuperVertex() PathVertex {
	return PathVertex{
		vertexType: _PATH_VERTEX_SENSOR_SUPER_VERTEX,
		alpha:      MakeConstantSpectrum(1),
	}
}

func validateSampledPathEdge(context *PathContext, pv, pvNext *PathVertex) {
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
		// TODO(akalin): Implement.

	case pv.vertexType == _PATH_VERTEX_SENSOR_VERTEX:
		// TODO(akalin): Implement.
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
			vertexType: _PATH_VERTEX_LIGHT_VERTEX,
			p:          p,
			pEpsilon:   pEpsilon,
			n:          n,
			alpha:      alphaNext,
			light:      light,
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
			vertexType: _PATH_VERTEX_SENSOR_VERTEX,
			p:          p,
			pEpsilon:   pEpsilon,
			n:          n,
			alpha:      alphaNext,
		}

	case _PATH_VERTEX_LIGHT_VERTEX:
		return false

	case _PATH_VERTEX_SENSOR_VERTEX:
		return false

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
	}
	return Spectrum{}
}
