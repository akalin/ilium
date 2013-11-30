package ilium

type ParticleTracer struct {
	russianRouletteState *RussianRouletteState
	maxEdgeCount         int
}

func (pt *ParticleTracer) InitializeParticleTracer(
	russianRouletteState *RussianRouletteState, maxEdgeCount int) {
	pt.russianRouletteState = russianRouletteState
	pt.maxEdgeCount = maxEdgeCount
}
