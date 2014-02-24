package ilium

import "fmt"
import "math/rand"
import "strings"

type PathContext struct {
	WeighingMethod           TracerWeighingMethod
	Beta                     float32
	RecordLightContributions bool
	ShouldDirectSampleLight  bool
	ShouldDirectSampleSensor bool
	RussianRouletteState     *RussianRouletteState
	LightBundle              SampleBundle
	SensorBundle             SampleBundle
	ChooseLightSample        Sample1D
	LightWiSamples           Sample2DArray
	SensorWiSamples          Sample2DArray
	DirectLighting1DSamples  []Sample1DArray
	DirectLighting2DSamples  []Sample2DArray
	DirectSensor1DSamples    []Sample1DArray
	DirectSensor2DSamples    []Sample2DArray
	Scene                    *Scene
	Sensor                   Sensor
	X, Y                     int
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

type pathVertexFlags int

const (
	_PATH_VERTEX_USES_DIRECT_LIGHTING pathVertexFlags = 1 << iota
)

func (flags pathVertexFlags) String() string {
	var flagStrings []string
	if (flags & _PATH_VERTEX_USES_DIRECT_LIGHTING) != 0 {
		flagStrings = append(flagStrings, "USES_DIRECT_LIGHTING")
	}

	return "{" + strings.Join(flagStrings, ", ") + "}"
}

type PathVertex struct {
	vertexType    pathVertexType
	transportType MaterialTransportType
	flags         pathVertexFlags
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
		flags:         pvPrev.flags,
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
		return fmt.Sprintf("{%v (%v), flags=%v, alpha=%v}",
			pv.vertexType, pv.transportType, pv.flags, pv.alpha)

	case _PATH_VERTEX_SENSOR_SUPER_VERTEX:
		return fmt.Sprintf("{%v (%v), flags=%v, alpha=%v}",
			pv.vertexType, pv.transportType, pv.flags, pv.alpha)

	case _PATH_VERTEX_LIGHT_VERTEX:
		return fmt.Sprintf(
			"{%v (%v), flags=%v, p=%v (e=%f), n=%v, alpha=%v, "+
				"pFromPrev=%f, gamma=%f, light=%v}",
			pv.vertexType, pv.transportType, pv.flags, pv.p,
			pv.pEpsilon, pv.n, pv.alpha, pv.pFromPrev,
			pv.gamma, pv.light)

	case _PATH_VERTEX_SENSOR_VERTEX:
		return fmt.Sprintf("{%v (%v), flags=%v, p=%v (e=%f), n=%v, "+
			"alpha=%v, pFromPrev=%f, gamma=%f}",
			pv.vertexType, pv.transportType, pv.flags, pv.p,
			pv.pEpsilon, pv.n, pv.alpha, pv.pFromPrev, pv.gamma)

	case _PATH_VERTEX_SURFACE_INTERACTION_VERTEX:
		return fmt.Sprintf("{%v (%v), flags=%v, p=%v (e=%f), n=%v, "+
			"alpha=%v, pFromPrev=%f, gamma=%f, light=%v, "+
			"sensor=%v, material=%v}",
			pv.vertexType, pv.transportType, pv.flags, pv.p,
			pv.pEpsilon, pv.n, pv.alpha, pv.pFromPrev, pv.gamma,
			pv.light, pv.sensor, pv.material)
	}

	return fmt.Sprintf("{%v}", pv.vertexType)
}

func validateSampledPathEdge(context *PathContext, pv, pvNext *PathVertex) {
	if pv != nil {
		if pv.transportType != pvNext.transportType {
			panic(fmt.Sprintf(
				"Sampled path edge with non-matching "+
					"transport types %v -> %v", pv, pvNext))
		}

		if pv.flags != pvNext.flags {
			panic(fmt.Sprintf(
				"Sampled path edge with non-matching "+
					"flags %v -> %v", pv, pvNext))
		}
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

	case pvPrev.IsSpecular(context) || pv.IsSpecular(context):
		return pvPrevGamma

	default:
		r := pFromNext / pv.pFromPrev
		return (1 + pvPrevGamma) * powFloat32(r, context.Beta)
	}
}

func (pv *PathVertex) IsSpecular(context *PathContext) bool {
	switch pv.vertexType {
	case _PATH_VERTEX_SENSOR_SUPER_VERTEX:
		return !context.RecordLightContributions ||
			context.Sensor.HasSpecularPosition()

	case _PATH_VERTEX_SENSOR_VERTEX:
		return !context.RecordLightContributions ||
			context.Sensor.HasSpecularDirection()
	}

	return false
}

func (pv *PathVertex) computePdfBackwardsSA(
	context *PathContext, pvPrev, pvNext *PathVertex) float32 {
	validateSampledPathEdge(context, pvPrev, pv)
	validateSampledPathEdge(context, pv, pvNext)
	// pvPrev not being a super-vertex means that it's a light,
	// sensor, or surface interaction vertex, and so if pv was
	// sampled from pvPrev, it must be just a surface interaction
	// vertex (and so must pvNext).
	if pvPrev.isSuperVertex() {
		panic(fmt.Sprintf("Super vertex %v", pvPrev))
	}
	if pv.vertexType != _PATH_VERTEX_SURFACE_INTERACTION_VERTEX {
		panic(fmt.Sprintf("Unexpected vertex %v", pv))
	}
	if pvNext.vertexType != _PATH_VERTEX_SURFACE_INTERACTION_VERTEX {
		panic(fmt.Sprintf("Unexpected vertex %v", pvNext))
	}

	var wo Vector3
	_ = wo.GetDirectionAndDistance(&pv.p, &pvPrev.p)
	var wi Vector3
	_ = wi.GetDirectionAndDistance(&pv.p, &pvNext.p)
	G := computeG(pv.p, pv.n, pvPrev.p, pvPrev.n)
	pdf := pv.material.ComputePdf(
		pv.transportType.AdjointType(), wi, wo, pv.n)
	return pdf * G
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
		var pFromPrevNext float32
		switch context.WeighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			pFromPrevNext = 1
		case TRACER_POWER_WEIGHTS:
			// If direct lighting is being used, we
			// account for it in the
			// _PATH_VERTEX_LIGHT_VERTEX case
			// below. (Also, subpaths ending at the light
			// vertex are always replaced with the
			// direct-lighting subpath.)
			pFromPrevNext = pChooseLight * pdfSpatial
		}
		*pvNext = PathVertex{
			vertexType:    _PATH_VERTEX_LIGHT_VERTEX,
			transportType: pv.transportType,
			flags:         pv.flags,
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
		var pFromPrevNext float32
		switch context.WeighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			pFromPrevNext = 1
		case TRACER_POWER_WEIGHTS:
			// TODO(akalin): Account for direct sensor sampling.
			pFromPrevNext = pdfSpatial
		}
		*pvNext = PathVertex{
			vertexType:    _PATH_VERTEX_SENSOR_VERTEX,
			transportType: pv.transportType,
			flags:         pv.flags,
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
		var pFromPrevNext float32
		switch context.WeighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			pFromPrevNext = 1
		case TRACER_POWER_WEIGHTS:
			if context.ShouldDirectSampleLight {
				// Adjust both spatial and directional
				// PDFs.

				var wi Vector3
				wi.Flip(&wo)
				pDirect := pv.light.ComputePdfFromPoint(
					intersection.P, intersection.PEpsilon,
					intersection.N, wi)
				if pDirect == 0 {
					// This may happen in rare cases.
					return false
				}

				pA := pv.pFromPrev

				// The spatial correction factor is
				// just p_D / p_A.
				pChooseLight :=
					context.Scene.ComputeLightPdf(pv.light)
				G := computeG(pv.p, pv.n,
					intersection.P, intersection.N)
				pv.pFromPrev = pChooseLight * pDirect * G

				// The directional correction factor
				// is p_A / p_D, so a factor of G
				// cancels out.
				pFromPrevNext = (pdfDirectional * pA) /
					(pChooseLight * pDirect)
			} else {
				G := computeG(pv.p, pv.n,
					intersection.P, intersection.N)
				pFromPrevNext = pdfDirectional * G
			}
		}
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
		var pFromPrevNext float32
		switch context.WeighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			pFromPrevNext = 1
		case TRACER_POWER_WEIGHTS:
			// TODO(akalin): Account for direct sensor sampling.
			G := computeG(
				pv.p, pv.n, intersection.P, intersection.N)
			extent := context.Sensor.GetExtent()
			pdfPixel := 1 / float32(extent.GetPixelCount())
			pFromPrevNext = pdfDirectional * G * pdfPixel
		}
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
		var pFromPrevNext float32
		switch context.WeighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			pFromPrevNext = 1
		case TRACER_POWER_WEIGHTS:
			G := computeG(
				pv.p, pv.n, intersection.P, intersection.N)
			pFromPrevNext = pdf * G
		}
		pvNext.initializeSurfaceInteractionVertex(
			context, pv, &intersection, alphaNext, pFromPrevNext)

	default:
		panic(fmt.Sprintf(
			"Unknown path vertex type %d", pv.vertexType))
	}

	validateSampledPathEdge(context, pv, pvNext)

	if pvPrev != nil && !pvPrev.isSuperVertex() {
		var pFromNextPrev float32
		switch context.WeighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			pFromNextPrev = 1
		case TRACER_POWER_WEIGHTS:
			pFromNextPrev = pv.computePdfBackwardsSA(
				context, pvPrev, pvNext)
		}
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

func (pv *PathVertex) SampleDirect(
	context *PathContext, k int, rng *rand.Rand,
	pvOther, pvNext *PathVertex) bool {
	switch pv.vertexType {
	case _PATH_VERTEX_LIGHT_SUPER_VERTEX:
		switch pvOther.vertexType {
		case _PATH_VERTEX_SENSOR_VERTEX:
			fallthrough

		case _PATH_VERTEX_SURFACE_INTERACTION_VERTEX:
			sampleIndex := k - 1
			u := context.DirectLighting1DSamples[0].GetSample(
				sampleIndex, rng)
			v := context.DirectLighting1DSamples[1].GetSample(
				sampleIndex, rng)
			w := context.DirectLighting2DSamples[0].GetSample(
				sampleIndex, rng)

			light, pChooseLight :=
				context.Scene.SampleLight(u.U)
			LeSpatialDivPdf, pdfDirect, p, pEpsilon, n :=
				light.SampleLeSpatialFromPoint(
					v.U, w.U1, w.U2, pvOther.p,
					pvOther.pEpsilon, pvOther.n)
			if LeSpatialDivPdf.IsBlack() || pdfDirect == 0 {
				return false
			}

			G := computeG(p, n, pvOther.p, pvOther.n)
			LeSpatialDivPdf.ScaleInv(
				&LeSpatialDivPdf, pChooseLight*G)

			albedo := &LeSpatialDivPdf
			var alphaNext Spectrum
			alphaNext.Mul(&pv.alpha, albedo)
			var pFromPrevNext float32
			switch context.WeighingMethod {
			case TRACER_UNIFORM_WEIGHTS:
				pFromPrevNext = 1
			case TRACER_POWER_WEIGHTS:
				// The spatial correction factor is
				// just p_D / p_A.
				pFromPrevNext = pChooseLight * pdfDirect * G
			}

			pv.flags |= _PATH_VERTEX_USES_DIRECT_LIGHTING
			*pvNext = PathVertex{
				vertexType:    _PATH_VERTEX_LIGHT_VERTEX,
				transportType: pv.transportType,
				flags:         pv.flags,
				p:             p,
				pEpsilon:      pEpsilon,
				n:             n,
				alpha:         alphaNext,
				pFromPrev:     pFromPrevNext,
				light:         light,
			}
			return true

		default:
			panic(fmt.Sprintf(
				"Invalid path vertex for direct sampling %v",
				pvOther))
		}

	case _PATH_VERTEX_SENSOR_SUPER_VERTEX:
		switch pvOther.vertexType {
		case _PATH_VERTEX_LIGHT_VERTEX:
			fallthrough

		case _PATH_VERTEX_SURFACE_INTERACTION_VERTEX:
			sampleIndex := k - 1
			u := context.DirectSensor1DSamples[0].GetSample(
				sampleIndex, rng)
			v := context.DirectSensor2DSamples[0].GetSample(
				sampleIndex, rng)

			WeSpatialDivPdf, pdfDirect, p, pEpsilon, n :=
				context.Sensor.SampleWeSpatialFromPoint(
					u.U, v.U1, v.U2, pvOther.p,
					pvOther.pEpsilon, pvOther.n)
			if WeSpatialDivPdf.IsBlack() || pdfDirect == 0 {
				return false
			}

			G := computeG(p, n, pvOther.p, pvOther.n)
			WeSpatialDivPdf.ScaleInv(&WeSpatialDivPdf, G)

			albedo := &WeSpatialDivPdf
			var alphaNext Spectrum
			alphaNext.Mul(&pv.alpha, albedo)
			var pFromPrevNext float32
			switch context.WeighingMethod {
			case TRACER_UNIFORM_WEIGHTS:
				pFromPrevNext = 1
			case TRACER_POWER_WEIGHTS:
				// TODO(akalin): Use real probabilities.
				//
				// TODO(akalin): Account for direct
				// sensor sampling.
				panic("Not implemented")
			}
			*pvNext = PathVertex{
				vertexType:    _PATH_VERTEX_SENSOR_VERTEX,
				transportType: pv.transportType,
				flags:         pv.flags,
				p:             p,
				pEpsilon:      pEpsilon,
				n:             n,
				alpha:         alphaNext,
				pFromPrev:     pFromPrevNext,
			}
			return true

		default:
			panic(fmt.Sprintf(
				"Invalid path vertex for direct sampling %v",
				pvOther))
		}

	default:
		panic(fmt.Sprintf(
			"Invalid path vertex for direct sampling %v", pv))
	}

	panic("Unexpectedly reached")
}

func validateConnectingLightVertex(context *PathContext, pv *PathVertex) {
	if pv.IsSpecular(context) {
		panic(fmt.Sprintf(
			"Invalid specular connecting light vertex %v", pv))
	}

	shouldUseDirectLighting := context.ShouldDirectSampleLight &&
		pv.vertexType == _PATH_VERTEX_LIGHT_VERTEX

	usesDirectLighting :=
		(pv.flags & _PATH_VERTEX_USES_DIRECT_LIGHTING) != 0

	if usesDirectLighting != shouldUseDirectLighting {
		panic(fmt.Sprintf(
			"Invalid connecting light vertex %v "+
				"(uses direct lighting = %t, expected %t)", pv,
			usesDirectLighting, shouldUseDirectLighting))
	}
}

func validateConnectingSensorVertex(context *PathContext, pv *PathVertex) {
	if pv.IsSpecular(context) {
		panic(fmt.Sprintf(
			"Invalid specular connecting sensor vertex %v", pv))
	}
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
		return

	case pv.vertexType == _PATH_VERTEX_SENSOR_VERTEX &&
		pvOther.vertexType == _PATH_VERTEX_LIGHT_VERTEX:
		return
	}

	panic(fmt.Sprintf("Invalid connection %v <-> %v", pv, pvOther))
}

func (pv *PathVertex) computeF(
	context *PathContext,
	x, y int, pvPrev *PathVertex, wi Vector3) Spectrum {
	switch pv.vertexType {
	case _PATH_VERTEX_LIGHT_VERTEX:
		return pv.light.ComputeLeDirectional(pv.p, pv.n, wi)

	case _PATH_VERTEX_SENSOR_VERTEX:
		return context.Sensor.ComputeWeDirectional(x, y, pv.p, pv.n, wi)

	case _PATH_VERTEX_SURFACE_INTERACTION_VERTEX:
		var wo Vector3
		_ = wo.GetDirectionAndDistance(&pv.p, &pvPrev.p)
		return pv.material.ComputeF(pv.transportType, wo, wi, pv.n)
	}

	panic("Unexpectedly reached")
}

func (pv *PathVertex) computeConnectingEdgeContribution(
	context *PathContext, x, y int,
	pvPrev, pvOther, pvOtherPrev *PathVertex) Spectrum {
	validateSampledPathEdge(context, pvPrev, pv)
	validateSampledPathEdge(context, pvOtherPrev, pvOther)
	validateConnectingPathEdge(context, pv, pvOther)

	var wi Vector3
	d := wi.GetDirectionAndDistance(&pv.p, &pvOther.p)
	shadowRay := Ray{
		pv.p, wi, pv.pEpsilon, d * (1.0 - pvOther.pEpsilon),
	}
	if context.Scene.Aggregate.Intersect(&shadowRay, nil) {
		return Spectrum{}
	}

	f := pv.computeF(context, x, y, pvPrev, wi)
	if f.IsBlack() {
		return Spectrum{}
	}

	var wiOther Vector3
	wiOther.Flip(&wi)
	fOther := pvOther.computeF(context, x, y, pvOtherPrev, wiOther)
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
}

// pv.vertexType >= pvOther.vertexType must hold.
func (pv *PathVertex) computeConnectionContribution(
	context *PathContext,
	pvPrev, pvOther, pvOtherPrev *PathVertex) (
	c Spectrum, contributionType TracerContributionType, x, y int) {
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
				return
			}

			var wo Vector3
			_ = wo.GetDirectionAndDistance(&pv.p, &pvPrev.p)
			c = pv.light.ComputeLe(pv.p, pv.n, wo)
			contributionType = TRACER_SENSOR_CONTRIBUTION
			x = context.X
			y = context.Y
			return

		case _PATH_VERTEX_SENSOR_SUPER_VERTEX:
			if pv.sensor == nil {
				return
			}

			var wo Vector3
			_ = wo.GetDirectionAndDistance(&pv.p, &pvPrev.p)
			x, y, c = pv.sensor.ComputePixelPositionAndWe(
				pv.p, pv.n, wo)
			contributionType = TRACER_LIGHT_CONTRIBUTION
			return

		case _PATH_VERTEX_SENSOR_VERTEX:
			contributionType = TRACER_LIGHT_CONTRIBUTION
			var wiOther Vector3
			_ = wiOther.GetDirectionAndDistance(&pvOther.p, &pv.p)
			var ok bool
			ok, x, y =
				context.Sensor.ComputePixelPosition(
					pvOther.p, pvOther.n, wiOther)
			if !ok {
				return
			}

		default:
			contributionType = TRACER_SENSOR_CONTRIBUTION
			x = context.X
			y = context.Y
		}

		c = pv.computeConnectingEdgeContribution(
			context, x, y, pvPrev, pvOther, pvOtherPrev)
		return

	case pv.vertexType == _PATH_VERTEX_SENSOR_VERTEX &&
		pvOther.vertexType == _PATH_VERTEX_LIGHT_VERTEX:

		contributionType = TRACER_LIGHT_CONTRIBUTION
		var wi Vector3
		_ = wi.GetDirectionAndDistance(&pv.p, &pvOther.p)
		var ok bool
		ok, x, y = context.Sensor.ComputePixelPosition(pv.p, pv.n, wi)
		if !ok {
			return
		}

		c = pv.computeConnectingEdgeContribution(
			context, x, y, pvPrev, pvOther, pvOtherPrev)
		return
	}

	panic("Unexpectedly reached")
	return
}

func (pv *PathVertex) ComputeUnweightedContribution(
	context *PathContext, pvPrev, pvOther, pvOtherPrev *PathVertex) (
	uC Spectrum, contributionType TracerContributionType,
	x, y int) {
	if pv.vertexType < pvOther.vertexType {
		pv, pvPrev, pvOther, pvOtherPrev =
			pvOther, pvOtherPrev, pv, pvPrev
	}

	c, contributionType, x, y := pv.computeConnectionContribution(
		context, pvPrev, pvOther, pvOtherPrev)

	if c.IsBlack() {
		contributionType = 0
		x = 0
		y = 0
		return
	}

	uC.Mul(&pv.alpha, &pvOther.alpha)
	uC.Mul(&uC, &c)
	return
}

func (pv *PathVertex) computeLeDirectionalPdf(
	context *PathContext, pvOther *PathVertex) float32 {
	var wo Vector3
	_ = wo.GetDirectionAndDistance(&pv.p, &pvOther.p)
	pDirectional := pv.light.ComputeLeDirectionalPdf(pv.p, pv.n, wo)

	if context.ShouldDirectSampleLight {
		var wi Vector3
		wi.Flip(&wo)
		pDirect := pv.light.ComputePdfFromPoint(
			pvOther.p, pvOther.pEpsilon, pvOther.n, wi)
		if pDirect == 0 {
			// This may happen in rare cases.
			return 0
		}

		pSurface := pv.light.ComputeLeSpatialPdf(pv.p)
		// The directional correction factor is p_A / p_D, so
		// the pChooseLight and G factors cancel out.
		return (pDirectional * pSurface) / pDirect
	}

	G := computeG(pv.p, pv.n, pvOther.p, pvOther.n)
	return pDirectional * G
}

func (pv *PathVertex) computeConnectionPdfBackwardsSA(
	context *PathContext, pvPrev, pvOther *PathVertex) float32 {
	validateSampledPathEdge(context, pvPrev, pv)
	if pv.vertexType >= pvOther.vertexType {
		validateConnectingPathEdge(context, pv, pvOther)
	} else {
		validateConnectingPathEdge(context, pvOther, pv)
	}
	// pvPrev not being a super-vertex means that it's a light,
	// sensor, or surface interaction vertex, and so if pv was
	// sampled from pvPrev, it must be just a surface interaction
	// vertex.
	if pvPrev.isSuperVertex() {
		panic(fmt.Sprintf("Super vertex %v", pvPrev))
	}
	if pv.vertexType != _PATH_VERTEX_SURFACE_INTERACTION_VERTEX {
		panic(fmt.Sprintf("Unexpected vertex %v", pv))
	}

	switch pvOther.vertexType {
	case _PATH_VERTEX_LIGHT_SUPER_VERTEX:
		if pv.light == nil {
			return 0
		}

		return pv.computeLeDirectionalPdf(context, pvPrev)

	case _PATH_VERTEX_SENSOR_SUPER_VERTEX:
		if pv.sensor == nil {
			return 0
		}

		var wo Vector3
		_ = wo.GetDirectionAndDistance(&pv.p, &pvPrev.p)

		// TODO(akalin): Account for direct sensor sampling.
		ok, x, y := pv.sensor.ComputePixelPosition(
			pv.p, pv.n, wo)
		if !ok {
			return 0
		}
		G := computeG(pv.p, pv.n, pvPrev.p, pvPrev.n)
		pdfDirectional := pv.sensor.ComputeWeDirectionalPdf(
			x, y, pv.p, pv.n, wo)
		extent := context.Sensor.GetExtent()
		pdfPixel := 1 / float32(extent.GetPixelCount())
		return pdfDirectional * G * pdfPixel

	case _PATH_VERTEX_LIGHT_VERTEX:
		fallthrough

	case _PATH_VERTEX_SENSOR_VERTEX:
		fallthrough

	case _PATH_VERTEX_SURFACE_INTERACTION_VERTEX:
		var wo Vector3
		_ = wo.GetDirectionAndDistance(&pv.p, &pvPrev.p)
		var wi Vector3
		_ = wi.GetDirectionAndDistance(&pv.p, &pvOther.p)
		G := computeG(pv.p, pv.n, pvPrev.p, pvPrev.n)
		pdf := pv.material.ComputePdf(
			pv.transportType.AdjointType(), wi, wo, pv.n)
		return pdf * G
	}

	panic("Unexpectedly reached")
}

func (pv *PathVertex) computeConnectionPdfForwardSA(
	context *PathContext, pvPrev,
	pvOther, pvOtherPrev *PathVertex) float32 {
	validateSampledPathEdge(context, pvPrev, pv)
	if pv.vertexType >= pvOther.vertexType {
		validateConnectingPathEdge(context, pv, pvOther)
	} else {
		validateConnectingPathEdge(context, pvOther, pv)
	}

	switch pv.vertexType {
	case _PATH_VERTEX_LIGHT_SUPER_VERTEX:
		if pvOther.vertexType !=
			_PATH_VERTEX_SURFACE_INTERACTION_VERTEX {
			panic("Unexpectedly reached")
		}

		if pvOther.light == nil {
			return 0
		}
		pChooseLight := context.Scene.ComputeLightPdf(pvOther.light)
		if context.ShouldDirectSampleLight {
			// The spatial correction factor is just
			// p_D / p_A.
			var wi Vector3
			_ = wi.GetDirectionAndDistance(
				&pvOtherPrev.p, &pvOther.p)
			pDirect := pvOther.light.ComputePdfFromPoint(
				pvOtherPrev.p, pvOtherPrev.pEpsilon,
				pvOtherPrev.n, wi)
			G := computeG(pvOther.p, pvOther.n,
				pvOtherPrev.p, pvOtherPrev.n)
			return pDirect * G * pChooseLight
		}

		pSpatial := pvOther.light.ComputeLeSpatialPdf(pvOther.p)
		return pChooseLight * pSpatial

	case _PATH_VERTEX_SENSOR_SUPER_VERTEX:
		if pvOther.vertexType !=
			_PATH_VERTEX_SURFACE_INTERACTION_VERTEX {
			panic("Unexpectedly reached")
		}

		// TODO(akalin): Account for direct sensor sampling.
		if pvOther.sensor == nil {
			return 0
		}
		pSpatial := pvOther.sensor.ComputeWeSpatialPdf(pvOther.p)
		return pSpatial

	case _PATH_VERTEX_LIGHT_VERTEX:
		return pv.computeLeDirectionalPdf(context, pvOther)

	case _PATH_VERTEX_SENSOR_VERTEX:
		// TODO(akalin): Account for direct sensor sampling.
		var wi Vector3
		_ = wi.GetDirectionAndDistance(&pv.p, &pvOther.p)
		ok, x, y := context.Sensor.ComputePixelPosition(pv.p, pv.n, wi)
		if !ok {
			return 0
		}
		G := computeG(pv.p, pv.n, pvOther.p, pvOther.n)
		pdfDirectional := context.Sensor.ComputeWeDirectionalPdf(
			x, y, pv.p, pv.n, wi)
		extent := context.Sensor.GetExtent()
		pdfPixel := 1 / float32(extent.GetPixelCount())
		return pdfDirectional * G * pdfPixel

	case _PATH_VERTEX_SURFACE_INTERACTION_VERTEX:
		switch pvOther.vertexType {
		case _PATH_VERTEX_LIGHT_VERTEX:
			fallthrough

		case _PATH_VERTEX_SENSOR_VERTEX:
			fallthrough

		case _PATH_VERTEX_SURFACE_INTERACTION_VERTEX:
			var wo Vector3
			_ = wo.GetDirectionAndDistance(&pv.p, &pvPrev.p)
			var wi Vector3
			_ = wi.GetDirectionAndDistance(&pv.p, &pvOther.p)
			G := computeG(pv.p, pv.n, pvOther.p, pvOther.n)
			pdf := pv.material.ComputePdf(
				pv.transportType, wo, wi, pv.n)
			return pdf * G
		}
	}

	panic("Unexpectedly reached")
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
		var pFromNextPrev float32
		switch context.WeighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			pFromNextPrev = 1
		case TRACER_POWER_WEIGHTS:
			pFromNextPrev = pv.computeConnectionPdfBackwardsSA(
				context, pvPrev, pvOther)
		}
		gammaPrev = pvPrev.computeGamma(
			context, pvPrevPrev, gammaPrevPrev, pFromNextPrev)
	}

	var gamma float32 = 0
	if !pv.isSuperVertex() {
		var pFromNext float32
		switch context.WeighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			pFromNext = 1
		case TRACER_POWER_WEIGHTS:
			pFromNext = pvOther.computeConnectionPdfForwardSA(
				context, pvOtherPrev, pv, pvPrev)
		}
		gamma = pv.computeGamma(context, pvPrev, gammaPrev, pFromNext)
	}

	return gamma
}

func (pv *PathVertex) ComputeWeight(
	context *PathContext,
	pvPrevPrev, pvPrev, pvOther,
	pvOtherPrev, pvOtherPrevPrev *PathVertex) float32 {
	validateConnectingLightVertex(context, pv)
	validateConnectingSensorVertex(context, pvOther)
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

func (pv *PathVertex) computeExpectedSubpathGamma(
	context *PathContext,
	pvAndPrevs, pvOtherAndPrevs []PathVertex) float32 {
	var rProd float32 = 1
	var gamma float32 = 0
	var pvPrev *PathVertex
	if len(pvAndPrevs) > 1 {
		pvPrev = &pvAndPrevs[len(pvAndPrevs)-2]
	}
	pvOther := &pvOtherAndPrevs[len(pvOtherAndPrevs)-1]
	var pvOtherPrev *PathVertex
	if len(pvOtherAndPrevs) > 1 {
		pvOtherPrev = &pvOtherAndPrevs[len(pvOtherAndPrevs)-2]
	}
	// Skip the super-vertex.
	for i := len(pvAndPrevs) - 1; i >= 1; i-- {
		v := &pvAndPrevs[i]
		vPrev := &pvAndPrevs[i-1]
		validateSampledPathEdge(context, vPrev, v)
		if v.IsSpecular(context) || vPrev.IsSpecular(context) {
			continue
		}
		var fromNext float32
		switch {
		case i == len(pvAndPrevs)-1:
			var pFromNext float32
			switch context.WeighingMethod {
			case TRACER_UNIFORM_WEIGHTS:
				pFromNext = 1
			case TRACER_POWER_WEIGHTS:
				pFromNext = pvOther.
					computeConnectionPdfForwardSA(
					context, pvOtherPrev, pv, pvPrev)
			}
			fromNext = pFromNext
		case i == len(pvAndPrevs)-2:
			var pFromNextPrev float32
			switch context.WeighingMethod {
			case TRACER_UNIFORM_WEIGHTS:
				pFromNextPrev = 1
			case TRACER_POWER_WEIGHTS:
				pFromNextPrev =
					pv.computeConnectionPdfBackwardsSA(
						context, pvPrev, pvOther)
			}
			fromNext = pFromNextPrev
		default:
			fromNext = v.pFromNext
		}
		r := fromNext / v.pFromPrev
		rProd *= powFloat32(r, context.Beta)
		gamma += rProd
	}
	return gamma
}

func (pv *PathVertex) ComputeExpectedWeight(
	context *PathContext, pvAndPrevs []PathVertex,
	pvOther *PathVertex, pvOtherAndPrevs []PathVertex) float32 {
	validateConnectingLightVertex(context, pv)
	validateConnectingSensorVertex(context, pvOther)
	if pv.vertexType >= pvOther.vertexType {
		validateConnectingPathEdge(context, pv, pvOther)
	} else {
		validateConnectingPathEdge(context, pvOther, pv)
	}

	gamma := pv.computeExpectedSubpathGamma(
		context, pvAndPrevs, pvOtherAndPrevs)
	gammaOther := pvOther.computeExpectedSubpathGamma(
		context, pvOtherAndPrevs, pvAndPrevs)

	return 1 / (gamma + 1 + gammaOther)
}
