package gateManager

func SensorUsed(id int) bool {
	SensorStructure.RLock()
	_, ok := SensorStructure.GateList[id]
	SensorStructure.RUnlock()
	return ok
}
