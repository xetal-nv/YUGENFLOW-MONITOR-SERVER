package gateManager

func SensorUsed(id int) bool {
	SensorList.RLock()
	_, ok := SensorList.GateList[id]
	SensorList.RUnlock()
	return ok
}
