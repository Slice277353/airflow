package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/g3n/engine/math32"
)

type SimulationData struct {
	Time            float32
	Acceleration    math32.Vector3
	WindPower       float32
	AngularMomentum math32.Vector3
	DampingEffect   float32
}

var simulationData []SimulationData

func recordSimulationData(dt float32, acceleration math32.Vector3, windPower float32, angularMomentum math32.Vector3, dampingEffect float32) {
	simulationData = append(simulationData, SimulationData{
		Time:            float32(time.Now().UnixNano()) / 1e9,
		Acceleration:    acceleration,
		WindPower:       windPower,
		AngularMomentum: angularMomentum,
		DampingEffect:   dampingEffect,
	})
}

func saveSimulationData() {
	filename := fmt.Sprintf("simulation_data_%d.json", time.Now().UnixNano())
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal("Error creating simulation data file: ", err)
	}
	defer file.Close()
	json.NewEncoder(file).Encode(simulationData)
}
