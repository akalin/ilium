package ilium

import "fmt"

type RussianRouletteMethod int

const (
	RUSSIAN_ROULETTE_FIXED        RussianRouletteMethod = iota
	RUSSIAN_ROULETTE_PROPORTIONAL RussianRouletteMethod = iota
)

type RussianRouletteState struct {
	method         RussianRouletteMethod
	startIndex     int
	maxProbability float32
	delta          float32
}

func MakeRussianRouletteState(
	config map[string]interface{}) *RussianRouletteState {
	var method RussianRouletteMethod
	methodConfig := config["russianRouletteMethod"].(string)
	switch methodConfig {
	case "fixed":
		method = RUSSIAN_ROULETTE_FIXED
	case "proportional":
		method = RUSSIAN_ROULETTE_PROPORTIONAL
	default:
		panic("unknown Russian roulette method " + methodConfig)
	}
	startIndex := int(config["russianRouletteStartIndex"].(float64))
	maxProbability :=
		float32(config["russianRouletteMaxProbability"].(float64))
	var delta float32
	if deltaConfig, ok := config["russianRouletteDelta"].(float64); ok {
		delta = float32(deltaConfig)
	} else {
		delta = 1
	}

	return &RussianRouletteState{
		method:         method,
		startIndex:     startIndex,
		maxProbability: maxProbability,
		delta:          delta,
	}
}

func (rrs *RussianRouletteState) IsLocalContinueProbabilityFixed() bool {
	return rrs.method == RUSSIAN_ROULETTE_FIXED
}

func (rrs *RussianRouletteState) IsContinueProbabilityFixed(i int) bool {
	return i < rrs.startIndex || rrs.IsLocalContinueProbabilityFixed()
}

func (rrs *RussianRouletteState) GetLocalContinueProbability(
	t *Spectrum) float32 {
	var pContinueRaw float32
	switch rrs.method {
	case RUSSIAN_ROULETTE_FIXED:
		pContinueRaw = rrs.maxProbability
	case RUSSIAN_ROULETTE_PROPORTIONAL:
		pContinueRaw = minFloat32(rrs.maxProbability, t.Y()/rrs.delta)
	default:
		panic(fmt.Sprintf("unknown Russian roulette method %d",
			rrs.method))
	}

	return minFloat32(1, maxFloat32(0, pContinueRaw))
}

func (rrs *RussianRouletteState) GetContinueProbability(
	i int, t *Spectrum) float32 {
	if i < rrs.startIndex {
		return 1
	}

	return rrs.GetLocalContinueProbability(t)
}
