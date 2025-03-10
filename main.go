package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/g3n/engine/app"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/light"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/renderer"
	"github.com/g3n/engine/util/helper"
	"github.com/g3n/engine/window"
)

// WindSource represents a wind generator in the scene
type WindSource struct {
	Position  math32.Vector3 // Wind source position
	Radius    float32        // Influence radius
	Speed     float32        // Wind speed
	Direction math32.Vector3 // Wind direction (normalized)
}

// WindParticle stores a particle's position and velocity
type WindParticle struct {
	Mesh     *graphic.Mesh  // The 3D particle
	Velocity math32.Vector3 // Movement direction
	Lifespan float32        // Time before disappearing
	Elapsed  float32        // Time passed
}

var scene *core.Node
var windParticles []*WindParticle

func createNumericInput(defaultValue float32, x, y float32, onChange func(value float32)) *gui.Edit {
	textInput := gui.NewEdit(100, fmt.Sprintf("%.2f", defaultValue)) // Width: 100px, default text
	textInput.SetPosition(x, y)

	textInput.Subscribe(gui.OnChange, func(name string, ev interface{}) {
		text := textInput.Text()

		// Remove invalid characters (allow digits, one dot, and optional negative sign at the start)
		filteredText := filterNumericInput(text)

		// Update text if filtering changed anything
		if text != filteredText {
			textInput.SetText(filteredText)
		}
	})

	textInput.Subscribe(gui.OnKeyDown, func(name string, ev interface{}) {
		kev := ev.(*window.KeyEvent)
		if kev.Key == window.KeyEnter {
			text := textInput.Text()
			if value, err := strconv.ParseFloat(text, 32); err == nil && value > 0 {
				onChange(float32(value))
			} else {
				textInput.SetText(fmt.Sprintf("%.2f", defaultValue)) // Reset to default if invalid
			}
		}
	})

	return textInput
}

// Helper function to filter numeric input
func filterNumericInput(input string) string {
	var builder strings.Builder
	dotCount := 0

	for i, char := range input {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		} else if char == '.' && dotCount == 0 { // Allow one decimal point
			builder.WriteRune(char)
			dotCount++
		} else if char == '-' && i == 0 { // Allow negative sign only at the start
			builder.WriteRune(char)
		}
	}

	return builder.String()
}

func createWindParticle(position, direction math32.Vector3) *WindParticle {
	particleGeom := geometry.NewSphere(0.05, 8, 8)
	particleMat := material.NewStandard(math32.NewColor("White"))
	particleMesh := graphic.NewMesh(particleGeom, particleMat)
	particleMesh.SetPositionVec(&position) // Use pointer

	scene.Add(particleMesh)

	return &WindParticle{
		Mesh:     particleMesh,
		Velocity: *direction.Clone().MultiplyScalar(0.5), // Move in wind direction
		Lifespan: 2.0,                                    // Lasts for 2 seconds
		Elapsed:  0,
	}
}

func updateWindParticles(deltaTime float32) {
	var newParticles []*WindParticle

	for _, particle := range windParticles {
		particle.Elapsed += deltaTime

		// Remove expired particles
		if particle.Elapsed >= particle.Lifespan {
			scene.Remove(particle.Mesh)
			continue
		}

		// Move particle
		pos := particle.Mesh.Position()
		pos.Add(&particle.Velocity)
		particle.Mesh.SetPositionVec(&pos)

		newParticles = append(newParticles, particle)
	}

	windParticles = newParticles
}

func main() {
	// Initialize app
	a := app.App()
	scene = core.NewNode()
	gui.Manager().Set(scene)

	// Physics variables
	velocity := math32.NewVector3(0, 0, 0)
	var dragCoefficient float32 = 0.47 // Approximate for a torus
	const airDensity = 1.225           // kg/m³ (standard air density)
	const area = 1.0                   // Simplified cross-sectional area
	var mass float32 = 1.0             // Mass of the torus
	const gravity = -9.8               // Gravity acceleration (m/s²)

	// Data for serialization
	type SimulationData struct {
		Time            float32
		Acceleration    math32.Vector3
		WindPower       float32
		AngularMomentum math32.Vector3
		DampingEffect   float32
	}
	var simulationData []SimulationData

	// Camera setup
	cam := camera.New(1)
	cam.SetPosition(0, 5, 10)
	cam.LookAt(&math32.Vector3{X: 0, Y: 0, Z: 0}, &math32.Vector3{X: 0, Y: 1, Z: 0})
	scene.Add(cam)
	camera.NewOrbitControl(cam)

	// Window resize handling
	onResize := func(evname string, ev interface{}) {
		width, height := a.GetSize()
		a.Gls().Viewport(0, 0, int32(width), int32(height))
		cam.SetAspect(float32(width) / float32(height))
	}
	a.Subscribe(window.OnWindowSize, onResize)
	onResize("", nil)

	// Create torus
	geom := geometry.NewTorus(1, 0.4, 12, 32, math32.Pi*2)
	mat := material.NewStandard(math32.NewColor("DarkBlue"))
	mesh := graphic.NewMesh(geom, mat)
	mesh.SetPosition(0, 1, 0)
	scene.Add(mesh)

	// Create surface
	surfaceGeom := geometry.NewPlane(20, 20)
	surfaceMat := material.NewStandard(math32.NewColor("Green"))
	surfaceMesh := graphic.NewMesh(surfaceGeom, surfaceMat)
	surfaceMesh.SetRotationX(-math32.Pi / 2)
	scene.Add(surfaceMesh)

	// Define wind sources
	windSources := []WindSource{
		{Position: *math32.NewVector3(2, 1, 0), Radius: 3.0, Speed: 10.0, Direction: *math32.NewVector3(-1, 0, 0).Normalize()},
		{Position: *math32.NewVector3(-2, 1, 0), Radius: 2.0, Speed: 5.0, Direction: *math32.NewVector3(1, 0, 0).Normalize()},
	}

	// Display wind sources as red spheres
	for _, wind := range windSources {
		sphereGeom := geometry.NewSphere(0.2, 16, 16)
		sphereMat := material.NewStandard(math32.NewColor("Red"))
		sphereMesh := graphic.NewMesh(sphereGeom, sphereMat)
		sphereMesh.SetPositionVec(&wind.Position)
		scene.Add(sphereMesh)
	}

	// UI: Wind Toggle Button
	windEnabled := false
	btn := gui.NewButton("Wind OFF")
	btn.SetPosition(100, 40)
	btn.SetSize(80, 40)
	btn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		windEnabled = !windEnabled
		if windEnabled {
			btn.Label.SetText("Wind ON")
		} else {
			btn.Label.SetText("Wind OFF")
		}
	})
	scene.Add(btn)

	massInput := createNumericInput(mass, 320, 100, func(value float32) {
		mass = value
	})
	massInput.SetPosition(100, 100)
	scene.Add(massInput)

	dragInput := createNumericInput(dragCoefficient, 320, 150, func(value float32) {
		dragCoefficient = value
	})
	dragInput.SetPosition(100, 150)
	scene.Add(dragInput)

	// Wind speed inputs for each wind source
	for i, wind := range windSources {
		windSpeedInput := createNumericInput(wind.Speed, 320, 200+float32(i*50), func(value float32) {
			windSources[i].Speed = value
		})
		windSpeedInput.SetPosition(100, 200+float32(i*50))
		scene.Add(windSpeedInput)
	}

	// Lights and helpers
	scene.Add(light.NewAmbient(&math32.Color{R: 1.0, G: 1.0, B: 1.0}, 0.8))
	pointLight := light.NewPoint(&math32.Color{R: 1, G: 1, B: 1}, 5.0)
	pointLight.SetPosition(1, 0, 2)
	scene.Add(pointLight)
	scene.Add(helper.NewAxes(1.0))

	// Background color
	a.Gls().ClearColor(0.5, 0.5, 0.5, 1.0)

	// Application loop
	a.Run(func(renderer *renderer.Renderer, deltaTime time.Duration) {
		a.Gls().Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)
		renderer.Render(scene, cam)

		torusPos := mesh.Position()

		if windEnabled {
			totalForce := math32.NewVector3(0, 0, 0)
			angularMomentum := math32.NewVector3(0, 0, 0)
			windPower := float32(0)
			dampingEffect := float32(0.01) // Reduced damping effect

			for i := range windSources {
				wind := &windSources[i]

				// Compute distance
				distanceVec := torusPos.Clone().Sub(&wind.Position)
				distance := distanceVec.Length()

				if distance <= wind.Radius {
					windVelocity := wind.Direction.Clone().MultiplyScalar(wind.Speed)
					dragMagnitude := 0.5 * airDensity * wind.Speed * wind.Speed * dragCoefficient * area
					dragForce := windVelocity.Clone().Normalize().MultiplyScalar(dragMagnitude)
					totalForce.Add(dragForce)

					// Calculate wind power absorption
					windPower += dragMagnitude * wind.Speed

					// Calculate angular momentum
					angularMomentum.Add(dragForce.Cross(&torusPos))

					// Add wind particles
					windParticles = append(windParticles, createWindParticle(wind.Position, wind.Direction))
				}
			}

			// Add gravity force
			gravityForce := math32.NewVector3(0, gravity*mass, 0)
			totalForce.Add(gravityForce)

			// Apply damping effect
			velocity.MultiplyScalar(1 - dampingEffect)

			dt := float32(deltaTime.Seconds())
			acceleration := totalForce.DivideScalar(mass)
			velocity.Add(acceleration.MultiplyScalar(dt))

			if velocity.Length() > 10 {
				velocity.Normalize().MultiplyScalar(10)
			}

			displacement := velocity.Clone().MultiplyScalar(dt)
			mesh.SetPositionVec(torusPos.Add(displacement))

			// Check for collision with surface
			if mesh.Position().Y < 1 {
				pos := mesh.Position()
				pos.SetY(1)
				mesh.SetPositionVec(&pos)
				velocity.SetY(0)
			}

			// Debug prints
			fmt.Printf("Position: %v, Velocity: %v, Total Force: %v\n", mesh.Position(), velocity, totalForce)

			// Collect data for serialization
			simulationData = append(simulationData, SimulationData{
				Time:            float32(time.Now().UnixNano()) / 1e9,
				Acceleration:    *acceleration,
				WindPower:       windPower,
				AngularMomentum: *angularMomentum,
				DampingEffect:   dampingEffect,
			})
		}

		// Update wind particles (move & remove expired)
		updateWindParticles(float32(deltaTime.Seconds()))
	})

	// Serialize data to JSON with a unique filename
	filename := fmt.Sprintf("simulation_data_%d.json", time.Now().UnixNano())
	file, _ := os.Create(filename)
	defer file.Close()
	json.NewEncoder(file).Encode(simulationData)
}
