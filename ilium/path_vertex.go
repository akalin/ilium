package ilium

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

type PathVertex struct{}

func MakeSensorSuperVertex() PathVertex {
	return PathVertex{}
}

func MakeLightSuperVertex() PathVertex {
	return PathVertex{}
}

func (pv *PathVertex) SampleNext(
	context *PathContext, i int, rng *rand.Rand,
	pvPrev, pvNext *PathVertex) bool {
	return false
}

func (pv *PathVertex) ComputeUnweightedContribution(
	context *PathContext,
	pvPrev, pvOther, pvOtherPrev *PathVertex) Spectrum {
	return Spectrum{}
}
