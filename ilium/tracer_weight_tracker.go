package ilium

import "fmt"

type TracerWeightTracker struct {
	beta         float32
	pVertexCount int
	lastPs       []float32
	qVertexCount int
	firstQs      []float32
}

// TODO(akalin): Keep track of middle q/p ratio.

// TracerWeightTracker objects are safely copyable, as long as only
// one copy is used for computing weights at a time.

func MakeTracerWeightTracker(beta float32) TracerWeightTracker {
	return TracerWeightTracker{beta: beta}
}

func (twt *TracerWeightTracker) AddP(vertexIndex int, p float32) {
	if vertexIndex < twt.pVertexCount-1 || vertexIndex > twt.pVertexCount {
		panic(fmt.Sprintf("Invalid p index %d for vertex count %d",
			vertexIndex, twt.pVertexCount))
	}

	if p < 0 || (p == 0 && vertexIndex == twt.pVertexCount) {
		panic(fmt.Sprintf("Invalid p value %f", p))
	}

	if vertexIndex == twt.pVertexCount-1 {
		twt.lastPs = append(twt.lastPs, p)
	} else {
		twt.lastPs = []float32{p}
		twt.pVertexCount++
	}
}

func (twt *TracerWeightTracker) AddQ(vertexIndex int, q float32) {
	if (vertexIndex == 0 && twt.qVertexCount > 1) ||
		(vertexIndex != 0 && vertexIndex != twt.qVertexCount) {
		panic(fmt.Sprintf("Invalid q index %d for vertex count %d",
			vertexIndex, twt.qVertexCount))
	}

	if q < 0 {
		panic(fmt.Sprintf("Invalid q value %f", q))
	}

	if vertexIndex == 0 {
		twt.firstQs = append(twt.firstQs, q)
		twt.qVertexCount = 1
	} else {
		twt.qVertexCount++
	}
}

func (twt *TracerWeightTracker) ComputeWeight(vertexCount int) float32 {
	if twt.pVertexCount != vertexCount {
		panic(fmt.Sprintf("p vertex count is %d, expected %d",
			twt.pVertexCount, vertexCount))
	}

	if twt.qVertexCount > 0 && twt.qVertexCount != vertexCount {
		panic(fmt.Sprintf("q vertex count is %d, expected %d",
			twt.qVertexCount, vertexCount))
	}

	var invW float32 = 1

	for i := 1; i < len(twt.lastPs); i++ {
		r := twt.lastPs[i] / twt.lastPs[0]
		invW += powFloat32(r, twt.beta)
	}

	for i := 0; i < len(twt.firstQs); i++ {
		r := twt.firstQs[i] / twt.lastPs[0]
		invW += powFloat32(r, twt.beta)
	}

	return 1 / invW
}
