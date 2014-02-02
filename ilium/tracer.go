package ilium

type TracerContributionType int

const (
	TRACER_SENSOR_CONTRIBUTION TracerContributionType = 1 << iota
	TRACER_LIGHT_CONTRIBUTION  TracerContributionType = 1 << iota
)

type TracerPathType int

const (
	TRACER_EMITTED_LIGHT_PATH      TracerPathType = 1 << iota
	TRACER_DIRECT_LIGHTING_PATH    TracerPathType = 1 << iota
	TRACER_EMITTED_IMPORTANCE_PATH TracerPathType = 1 << iota
	TRACER_DIRECT_SENSOR_PATH      TracerPathType = 1 << iota
)

func (pathTypes TracerPathType) GetContributionTypes() TracerContributionType {
	var contributionTypes TracerContributionType

	if ((pathTypes & TRACER_EMITTED_LIGHT_PATH) != 0) ||
		((pathTypes & TRACER_DIRECT_LIGHTING_PATH) != 0) {
		contributionTypes |= TRACER_SENSOR_CONTRIBUTION
	}

	if ((pathTypes & TRACER_EMITTED_IMPORTANCE_PATH) != 0) ||
		((pathTypes & TRACER_DIRECT_SENSOR_PATH) != 0) {
		contributionTypes |= TRACER_LIGHT_CONTRIBUTION
	}

	return contributionTypes
}

func MakeTracerPathType(pathTypeString string) TracerPathType {
	switch pathTypeString {
	case "emittedLight":
		return TRACER_EMITTED_LIGHT_PATH
	case "directLighting":
		return TRACER_DIRECT_LIGHTING_PATH
	case "emittedImportance":
		return TRACER_EMITTED_IMPORTANCE_PATH
	case "directSensor":
		return TRACER_DIRECT_SENSOR_PATH
	default:
		panic("unknown path type " + pathTypeString)
	}
}
