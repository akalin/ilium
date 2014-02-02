package ilium

import "fmt"

type TracerContributionType int

const (
	TRACER_SENSOR_CONTRIBUTION TracerContributionType = 1 << iota
	TRACER_LIGHT_CONTRIBUTION  TracerContributionType = 1 << iota
)

func (contributionTypes TracerContributionType) HasContributions(
	contributions TracerContributionType) bool {
	return (contributionTypes & contributions) == contributions
}

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

func (pathTypes TracerPathType) HasContributions(
	contributions TracerContributionType) bool {
	return pathTypes.GetContributionTypes().HasContributions(contributions)
}

func (pathTypes TracerPathType) HasPaths(paths TracerPathType) bool {
	return (pathTypes & paths) == paths
}

func (pathTypes TracerPathType) HasAlternatePath(
	alternatePathType TracerPathType, edgeCount int, sensor Sensor) bool {
	if !pathTypes.HasPaths(alternatePathType) {
		return false
	}

	switch alternatePathType {
	case TRACER_EMITTED_LIGHT_PATH:
		return true

	case TRACER_DIRECT_LIGHTING_PATH:
		// Direct lighting isn't done with the first edge.
		return edgeCount > 1

	case TRACER_EMITTED_IMPORTANCE_PATH:
		return !sensor.HasSpecularPosition()

	case TRACER_DIRECT_SENSOR_PATH:
		return !sensor.HasSpecularDirection()

	default:
		panic(fmt.Sprintf(
			"unknown alternate path type %d", alternatePathType))
	}
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

type TracerWeighingMethod int

const (
	TRACER_UNIFORM_WEIGHTS TracerWeighingMethod = iota
	TRACER_POWER_WEIGHTS   TracerWeighingMethod = iota
)

func MakeTracerWeighingMethod(weighingMethodString string) (
	weighingMethod TracerWeighingMethod, beta float32) {
	switch weighingMethodString {
	case "uniform":
		weighingMethod = TRACER_UNIFORM_WEIGHTS
		beta = 1
		return
	case "balanced":
		weighingMethod = TRACER_POWER_WEIGHTS
		beta = 1
		return
	case "power":
		weighingMethod = TRACER_POWER_WEIGHTS
		beta = 2
		return
	default:
		panic("unknown weighing method " + weighingMethodString)
	}
}
