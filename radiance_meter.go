package main

type RadianceMeter struct{}

func MakeRadianceMeter() *RadianceMeter {
	return &RadianceMeter{}
}

func (rm *RadianceMeter) GenerateRay(sensorSample SensorSample) Ray {
	return Ray{}
}

func (rm *RadianceMeter) RecordSample(sensorSample SensorSample, Li Spectrum) {
}

func (rm *RadianceMeter) EmitSignal() {
}
