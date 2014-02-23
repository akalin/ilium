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

func (t pathVertexType) String() string {
	switch t {
	case _PATH_VERTEX_LIGHT_SUPER_VERTEX:
		return "LIGHT_SUPER_VERTEX"

	case _PATH_VERTEX_SENSOR_SUPER_VERTEX:
		return "SENSOR_SUPER_VERTEX"

	case _PATH_VERTEX_LIGHT_VERTEX:
		return "LIGHT_VERTEX"

	case _PATH_VERTEX_SENSOR_VERTEX:
		return "SENSOR_VERTEX"

	case _PATH_VERTEX_SURFACE_INTERACTION_VERTEX:
		return "SURFACE_INTERACTION_VERTEX"
	}

	return "<Unknown path vertex type>"
}

type PathVertex struct {
	vertexType    pathVertexType
	transportType MaterialTransportType
	p             Point3
	pEpsilon      float32
	n             Normal3
	alpha         Spectrum
	// Used for (incremental and non-incremental) weight
	// computations.
	pFromPrev float32
	pFromNext float32
	// Used for incremental weight computation.
	gamma float32
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
		pFromPrev:     1,
	}
}

func MakeSensorSuperVertex() PathVertex {
	return PathVertex{
		vertexType:    _PATH_VERTEX_SENSOR_SUPER_VERTEX,
		transportType: MATERIAL_LIGHT_TRANSPORT,
		alpha:         MakeConstantSpectrum(1),
		pFromPrev:     1,
	}
}

func (pv *PathVertex) isSuperVertex() bool {
	return pv.vertexType == _PATH_VERTEX_LIGHT_SUPER_VERTEX ||
		pv.vertexType == _PATH_VERTEX_SENSOR_SUPER_VERTEX
}

func (pv *PathVertex) initializeSurfaceInteractionVertex(
	context *PathContext, pvPrev *PathVertex, intersection *Intersection,
	alpha Spectrum, pFromPrev float32) {
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
		pFromPrev:     pFromPrev,
		light:         intersection.Light,
		sensor:        sensor,
		material:      intersection.Material,
	}
}

func (pv *PathVertex) String() string {
	switch pv.vertexType {
	case _PATH_VERTEX_LIGHT_SUPER_VERTEX:
		return fmt.Sprintf("{%v (%v), alpha=%v}",
			pv.vertexType, pv.transportType, pv.alpha)

	case _PATH_VERTEX_SENSOR_SUPER_VERTEX:
		return fmt.Sprintf("{%v (%v), alpha=%v}",
			pv.vertexType, pv.transportType, pv.alpha)

	case _PATH_VERTEX_LIGHT_VERTEX:
		return fmt.Sprintf(
			"{%v (%v), p=%v (e=%f), n=%v, alpha=%v, "+
				"pFromPrev=%f, gamma=%f, light=%v}",
			pv.vertexType, pv.transportType, pv.p, pv.pEpsilon,
			pv.n, pv.alpha, pv.pFromPrev, pv.gamma, pv.light)

	case _PATH_VERTEX_SENSOR_VERTEX:
		return fmt.Sprintf("{%v (%v), p=%v (e=%f), n=%v, alpha=%v, "+
			"pFromPrev=%f, gamma=%f}",
			pv.vertexType, pv.transportType, pv.p, pv.pEpsilon,
			pv.n, pv.alpha, pv.pFromPrev, pv.gamma)

	case _PATH_VERTEX_SURFACE_INTERACTION_VERTEX:
		return fmt.Sprintf("{%v (%v), p=%v (e=%f), n=%v, alpha=%v, "+
			"pFromPrev=%f, gamma=%f, light=%v, sensor=%v, "+
			"material=%v}",
			pv.vertexType, pv.transportType, pv.p, pv.pEpsilon,
			pv.n, pv.alpha, pv.pFromPrev, pv.gamma, pv.light,
			pv.sensor, pv.material)
	}

	return fmt.Sprintf("{%v}", pv.vertexType)
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

func (pv *PathVertex) computeGamma(
	context *PathContext, pvPrev *PathVertex,
	pvPrevGamma, pFromNext float32) float32 {
	validateSampledPathEdge(context, pvPrev, pv)
	if pv.isSuperVertex() {
		panic(fmt.Sprintf("Super vertex %v", pv))
	}
	if !isFiniteFloat32(pv.pFromPrev) || pv.pFromPrev <= 0 {
		panic(fmt.Sprintf("Invalid pFromPrev value for %v", pv))
	}
	if !isFiniteFloat32(pFromNext) || pFromNext < 0 {
		panic(fmt.Sprintf("Invalid pFromNext value %f", pFromNext))
	}

	switch {
	case pv.vertexType == _PATH_VERTEX_LIGHT_SUPER_VERTEX ||
		pv.vertexType == _PATH_VERTEX_SENSOR_SUPER_VERTEX:
		return 0

	// TODO(akalin): Generalize this into a specularity check for
	// pvPrev and pv.
	case (pvPrev.vertexType == _PATH_VERTEX_SENSOR_SUPER_VERTEX ||
		pvPrev.vertexType == _PATH_VERTEX_SENSOR_VERTEX) ||
		(pv.vertexType == _PATH_VERTEX_SENSOR_SUPER_VERTEX ||
			pv.vertexType == _PATH_VERTEX_SENSOR_VERTEX):
		return pvPrevGamma

	default:
		return (1 + pvPrevGamma) * (pFromNext / pv.pFromPrev)
	}
}

func (pv *PathVertex) SampleNext(
	context *PathContext, i int, rng *rand.Rand,
	pvPrevPrev, pvPrev, pvNext *PathVertex) bool {
	if pvPrev != nil {
		validateSampledPathEdge(context, pvPrevPrev, pvPrev)
	}
	validateSampledPathEdge(context, pvPrev, pv)

	switch pv.vertexType {
	case _PATH_VERTEX_LIGHT_SUPER_VERTEX:
		light, pChooseLight :=
			context.Scene.SampleLight(context.ChooseLightSample.U)
		p, pEpsilon, n, LeSpatialDivPdf, pdfSpatial :=
			light.SampleSurface(context.LightBundle)
		if LeSpatialDivPdf.IsBlack() || pdfSpatial == 0 {
			return false
		}

		LeSpatialDivPdf.ScaleInv(&LeSpatialDivPdf, pChooseLight)

		albedo := &LeSpatialDivPdf
		if !pv.shouldContinue(context, i, albedo, rng) {
			return false
		}

		var alphaNext Spectrum
		alphaNext.Mul(&pv.alpha, albedo)
		// TODO(akalin): Use real probabilities.
		var pFromPrevNext float32 = 1
		*pvNext = PathVertex{
			vertexType:    _PATH_VERTEX_LIGHT_VERTEX,
			transportType: pv.transportType,
			p:             p,
			pEpsilon:      pEpsilon,
			n:             n,
			alpha:         alphaNext,
			pFromPrev:     pFromPrevNext,
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
		// TODO(akalin): Use real probabilities.
		var pFromPrevNext float32 = 1
		*pvNext = PathVertex{
			vertexType:    _PATH_VERTEX_SENSOR_VERTEX,
			transportType: pv.transportType,
			p:             p,
			pEpsilon:      pEpsilon,
			n:             n,
			alpha:         alphaNext,
			pFromPrev:     pFromPrevNext,
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
		// TODO(akalin): Use real probabilities.
		var pFromPrevNext float32 = 1
		pvNext.initializeSurfaceInteractionVertex(
			context, pv, &intersection, alphaNext, pFromPrevNext)

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
		// TODO(akalin): Use real probabilities.
		var pFromPrevNext float32 = 1
		pvNext.initializeSurfaceInteractionVertex(
			context, pv, &intersection, alphaNext, pFromPrevNext)

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
		// TODO(akalin): Use real probabilities.
		var pFromPrevNext float32 = 1
		pvNext.initializeSurfaceInteractionVertex(
			context, pv, &intersection, alphaNext, pFromPrevNext)

	default:
		panic(fmt.Sprintf(
			"Unknown path vertex type %d", pv.vertexType))
	}

	validateSampledPathEdge(context, pv, pvNext)

	if pvPrev != nil && !pvPrev.isSuperVertex() {
		// TODO(akalin): Use real probabilities.
		var pFromNextPrev float32 = 1
		var pvPrevPrevGamma float32 = 0
		if pvPrevPrev != nil {
			pvPrevPrevGamma = pvPrevPrev.gamma
		}
		pvPrev.pFromNext = pFromNextPrev
		pvPrev.gamma = pvPrev.computeGamma(
			context, pvPrevPrev, pvPrevPrevGamma, pFromNextPrev)
	}
	return true
}

func validateConnectingPathEdge(context *PathContext, pv, pvOther *PathVertex) {
	if pv.vertexType < pvOther.vertexType {
		panic(fmt.Sprintf(
			"Invalid connection order (%v, %v)", pv, pvOther))
	}

	if pv.transportType == pvOther.transportType {
		panic(fmt.Sprintf("Invalid connection %v <-> %v with the same "+
			"transport type", pv, pvOther))
	}

	switch {
	case pv.vertexType == _PATH_VERTEX_SURFACE_INTERACTION_VERTEX:
		if pvOther.vertexType != _PATH_VERTEX_SENSOR_SUPER_VERTEX &&
			pvOther.vertexType != _PATH_VERTEX_SENSOR_VERTEX {
			return
		}

	case pv.vertexType == _PATH_VERTEX_SENSOR_VERTEX &&
		pvOther.vertexType == _PATH_VERTEX_LIGHT_VERTEX:
		// TODO(akalin): Support.
		break
	}

	panic(fmt.Sprintf("Invalid connection %v <-> %v", pv, pvOther))
}

func (pv *PathVertex) computeF(pvPrev *PathVertex, wi Vector3) Spectrum {
	switch pv.vertexType {
	case _PATH_VERTEX_LIGHT_VERTEX:
		return pv.light.ComputeLeDirectional(pv.p, pv.n, wi)

	case _PATH_VERTEX_SENSOR_VERTEX:
		// TODO(akalin): Implement.

	case _PATH_VERTEX_SURFACE_INTERACTION_VERTEX:
		var wo Vector3
		_ = wo.GetDirectionAndDistance(&pv.p, &pvPrev.p)
		return pv.material.ComputeF(pv.transportType, wo, wi, pv.n)
	}

	panic("Not implemented")
}

// pv.vertexType >= pvOther.vertexType must hold.
func (pv *PathVertex) computeConnectionContribution(
	context *PathContext,
	pvPrev, pvOther, pvOtherPrev *PathVertex) Spectrum {
	validateSampledPathEdge(context, pvPrev, pv)
	validateSampledPathEdge(context, pvOtherPrev, pvOther)
	validateConnectingPathEdge(context, pv, pvOther)

	// Since pv.vertexType >= pvOther.vertexType and the
	// connection contribution is symmetric, this reduces the
	// number of cases to check.
	switch {
	case pv.vertexType == _PATH_VERTEX_SURFACE_INTERACTION_VERTEX:
		switch pvOther.vertexType {
		case _PATH_VERTEX_LIGHT_SUPER_VERTEX:
			if pv.light == nil {
				return Spectrum{}
			}
			var wo Vector3
			_ = wo.GetDirectionAndDistance(&pv.p, &pvPrev.p)
			return pv.light.ComputeLe(pv.p, pv.n, wo)

		case _PATH_VERTEX_SENSOR_SUPER_VERTEX:
			// TODO(akalin): Implement.
			panic("Not implemented")
			return Spectrum{}
		}

		// The rest of the cases are handled below.

		var wi Vector3
		d := wi.GetDirectionAndDistance(&pv.p, &pvOther.p)
		shadowRay := Ray{
			pv.p, wi, pv.pEpsilon, d * (1.0 - pvOther.pEpsilon),
		}
		if context.Scene.Aggregate.Intersect(&shadowRay, nil) {
			return Spectrum{}
		}

		f := pv.computeF(pvPrev, wi)
		if f.IsBlack() {
			return Spectrum{}
		}

		var wiOther Vector3
		wiOther.Flip(&wi)
		fOther := pvOther.computeF(pvOtherPrev, wiOther)
		if fOther.IsBlack() {
			return Spectrum{}
		}

		G := computeG(pv.p, pv.n, pvOther.p, pvOther.n)
		if G == 0 {
			return Spectrum{}
		}

		var c Spectrum
		c.Mul(&f, &fOther)
		c.Scale(&c, G)
		return c

	case pv.vertexType == _PATH_VERTEX_SENSOR_VERTEX &&
		pvOther.vertexType == _PATH_VERTEX_LIGHT_VERTEX:
		// TODO(akalin): Implement.
		panic("Not implemented")
	}

	panic("Unexpectedly reached")
	return Spectrum{}
}

func (pv *PathVertex) ComputeUnweightedContribution(
	context *PathContext,
	pvPrev, pvOther, pvOtherPrev *PathVertex) Spectrum {
	var c Spectrum
	if pv.vertexType >= pvOther.vertexType {
		c = pv.computeConnectionContribution(
			context, pvPrev, pvOther, pvOtherPrev)
	} else {
		c = pvOther.computeConnectionContribution(
			context, pvOtherPrev, pv, pvPrev)
	}

	if c.IsBlack() {
		return Spectrum{}
	}

	var uC Spectrum
	uC.Mul(&pv.alpha, &pvOther.alpha)
	uC.Mul(&uC, &c)
	return uC
}

func (pv *PathVertex) computeSubpathGamma(context *PathContext,
	pvPrevPrev, pvPrev, pvOther, pvOtherPrev *PathVertex) float32 {
	if pvPrev != nil {
		validateSampledPathEdge(context, pvPrevPrev, pvPrev)
	}
	validateSampledPathEdge(context, pvPrev, pv)

	var gammaPrevPrev float32 = 0
	if pvPrevPrev != nil {
		gammaPrevPrev = pvPrevPrev.gamma
	}

	var gammaPrev float32 = 0
	if pvPrev != nil && !pvPrev.isSuperVertex() {
		// TODO(akalin): Use real probabilities.
		var pFromNextPrev float32 = 1
		gammaPrev = pvPrev.computeGamma(
			context, pvPrevPrev, gammaPrevPrev, pFromNextPrev)
	}

	var gamma float32 = 0
	if !pv.isSuperVertex() {
		// TODO(akalin): Use real probabilities.
		var pFromNext float32 = 1
		gamma = pv.computeGamma(context, pvPrev, gammaPrev, pFromNext)
	}

	return gamma
}

func (pv *PathVertex) ComputeWeight(
	context *PathContext,
	pvPrevPrev, pvPrev, pvOther,
	pvOtherPrev, pvOtherPrevPrev *PathVertex) float32 {
	if pv.vertexType >= pvOther.vertexType {
		validateConnectingPathEdge(context, pv, pvOther)
	} else {
		validateConnectingPathEdge(context, pvOther, pv)
	}

	gamma := pv.computeSubpathGamma(
		context, pvPrevPrev, pvPrev, pvOther, pvOtherPrev)
	gammaOther := pvOther.computeSubpathGamma(
		context, pvOtherPrevPrev, pvOtherPrev, pv, pvPrev)

	return 1 / (gamma + 1 + gammaOther)
}
