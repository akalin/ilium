package main

import "fmt"

type DoubleCheckPrimitive struct {
	primary, secondary Primitive
	epsilon            float32
}

func (dcp *DoubleCheckPrimitive) Intersect(
	ray *Ray, intersection *Intersection) bool {
	var primaryIntersection, secondaryIntersection Intersection
	primaryFound := dcp.primary.Intersect(ray, &primaryIntersection)
	secondaryFound := dcp.secondary.Intersect(ray, &secondaryIntersection)
	if primaryFound != secondaryFound {
		fmt.Printf(
			"ray=%v: primary found intersection: %v, "+
				"secondary found intersection: %v\n",
			*ray, primaryFound, secondaryFound)
	}
	if !primaryIntersection.IsCloseTo(&secondaryIntersection, dcp.epsilon) {
		fmt.Printf(
			"ray=%v: primary intersection: %v, "+
				"secondary intersection: %v\n",
			*ray, primaryIntersection, secondaryIntersection)
	}
	*intersection = primaryIntersection
	return primaryFound
}

func (dcp *DoubleCheckPrimitive) GetBoundingBox() BBox {
	primaryBoundingBox := dcp.primary.GetBoundingBox()
	secondaryBoundingBox := dcp.secondary.GetBoundingBox()
	if primaryBoundingBox != secondaryBoundingBox {
		fmt.Printf(
			"primary bounding box: %v, "+
				"secondary bounding box: %v",
			primaryBoundingBox, secondaryBoundingBox)
	}
	return primaryBoundingBox
}

func (dcp *DoubleCheckPrimitive) GetSensors() []Sensor {
	primarySensors := dcp.primary.GetSensors()
	secondarySensors := dcp.secondary.GetSensors()
	if len(primarySensors) != len(secondarySensors) {
		fmt.Printf(
			"primary has %d sensor(s), "+
				"secondary has %d sensor(s)\n",
			len(primarySensors), len(secondarySensors))
	}
	return primarySensors
}

func MakeDoubleCheckPrimitive(
	config map[string]interface{}, primitives []Primitive) Aggregate {
	primaryConfig := config["primary"].(map[string]interface{})
	secondaryConfig := config["secondary"].(map[string]interface{})
	epsilon := float32(config["epsilon"].(float64))
	primary := MakeAggregateWithPrimitives(primaryConfig, primitives)
	secondary := MakeAggregateWithPrimitives(secondaryConfig, primitives)
	return &DoubleCheckPrimitive{primary, secondary, epsilon}
}
