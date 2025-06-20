package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

const (
	airDensity       = 1.225
	dragCoefficient  = 0.47
	area             = 1.0
	gravity          = -9.8
	buoyancyFactor   = 0.1
	turbulenceFactor = 0.5
	thermalDiffusion = 0.02
)

type ParticleData struct {
	Position struct {
		X, Y, Z float32
	}
	Velocity struct {
		X, Y, Z float32
	}
	Temperature float32
}

type SimulationSnapshot struct {
	Timestamp   float64
	Particles   []ParticleData
	WindSources []WindSource
}

var (
	simulationHistory []SimulationSnapshot
	startTime         float64
	isRecording       bool
)

func recordSimulationFrame() {
	if !windEnabled || !isRecording {
		return
	}

	currentTime := float64(time.Now().UnixNano()) / 1e9
	if len(simulationHistory) == 0 {
		startTime = currentTime
		log.Printf("Starting data collection at time: %.2f", startTime)
	}

	// Ensure we have a reasonable time delta (at least 0.016s = ~60fps)
	if len(simulationHistory) > 0 {
		lastSnapshot := simulationHistory[len(simulationHistory)-1]
		if (currentTime-startTime)-lastSnapshot.Timestamp < 0.016 {
			return // Skip if not enough time has passed
		}
	}

	snapshot := SimulationSnapshot{
		Timestamp: currentTime - startTime,
	}

	// Record ALL particles, both wind and fluid
	allParticles := make([]ParticleData, 0)

	// Record wind particles
	windCount := 0
	for _, p := range windParticles {
		if p != nil && p.Alive {
			particleData := ParticleData{
				Position: struct{ X, Y, Z float32 }{
					X: p.Position.X,
					Y: p.Position.Y,
					Z: p.Position.Z,
				},
				Velocity: struct{ X, Y, Z float32 }{
					X: p.Velocity.X,
					Y: p.Velocity.Y,
					Z: p.Velocity.Z,
				},
				Temperature: p.Temperature,
			}
			allParticles = append(allParticles, particleData)
			windCount++
		}
	}

	// Record fluid particles
	fluidCount := 0
	for _, p := range fluidParticles {
		if p.Mesh != nil {
			particleData := ParticleData{
				Position: struct{ X, Y, Z float32 }{
					X: p.X,
					Y: p.Y,
					Z: p.Z,
				},
				Velocity: struct{ X, Y, Z float32 }{
					X: p.VX,
					Y: p.VY,
					Z: p.VZ,
				},
				Temperature: 20.0,
			}
			allParticles = append(allParticles, particleData)
			fluidCount++
		}
	}

	// Only store snapshot if we have particles
	if len(allParticles) > 0 {
		snapshot.Particles = allParticles
		simulationHistory = append(simulationHistory, snapshot)

		// Log every 30th frame to reduce output
		if len(simulationHistory)%30 == 0 {
			log.Printf("Recording frame %d: t=%.2fs, Wind particles: %d, Fluid particles: %d, Total: %d",
				len(simulationHistory), snapshot.Timestamp, windCount, fluidCount, len(allParticles))
		}
	}
}

func calculateAverageDragForce() float32 {
	if len(windParticles) == 0 {
		log.Printf("No wind particles for drag force calculation")
		return 0
	}
	var totalForce float32
	for _, p := range windParticles {
		if p != nil && p.Alive {
			velocity := p.Velocity.Length()
			// Defensive: if Mass is zero, set to 1
			mass := p.Mass
			if mass == 0 {
				mass = 1.0
			}
			totalForce += 0.5 * airDensity * dragCoefficient * area * velocity * velocity
		}
	}
	avg := totalForce / float32(len(windParticles))
	log.Printf("Calculated average drag force: %f N", avg)
	return avg
}

func calculateAverageLiftForce() float32 {
	if len(windParticles) == 0 {
		log.Printf("No wind particles for lift force calculation")
		return 0
	}
	var totalForce float32
	for _, p := range windParticles {
		if p != nil && p.Alive {
			mass := p.Mass
			if mass == 0 {
				mass = 1.0
			}
			totalForce += mass * buoyancyFactor * (p.Temperature - 20.0)
		}
	}
	avg := totalForce / float32(len(windParticles))
	log.Printf("Calculated average lift force: %f N", avg)
	return avg
}

func startRecording() {
	simulationHistory = nil // Clear any existing history
	isRecording = true
	log.Printf("Started recording simulation data")
}

func stopRecording() {
	isRecording = false
	log.Printf("Stopped recording. Total frames captured: %d", len(simulationHistory))
}

func saveSimulationData() (string, error) {
	if len(simulationHistory) < 2 {
		return "", fmt.Errorf("insufficient simulation data: need at least 2 snapshots, got %d", len(simulationHistory))
	}

	filename := fmt.Sprintf("simulation_data_%d.json", time.Now().UnixNano())
	// Save in current working directory
	filepath := filename

	// Print summary before saving
	log.Printf("\nSaving simulation data:")
	log.Printf("Total frames: %d", len(simulationHistory))
	log.Printf("Time range: %.2fs to %.2fs",
		simulationHistory[0].Timestamp,
		simulationHistory[len(simulationHistory)-1].Timestamp)

	for i, snapshot := range simulationHistory {
		if i < 3 || i > len(simulationHistory)-3 { // Print first and last few frames
			log.Printf("Frame %d: t=%.2fs, Particles: %d",
				i, snapshot.Timestamp, len(snapshot.Particles))
		} else if i == 3 {
			log.Printf("...")
		}
	}

	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(simulationHistory); err != nil {
		return "", fmt.Errorf("error encoding data: %v", err)
	}

	log.Printf("Successfully saved simulation data to %s", filepath)
	return filepath, nil
}
