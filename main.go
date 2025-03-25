package main

import (
	"log"
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

var scene *core.Node
var mesh *core.Node
var windEnabled bool

func main() {
	a := app.App()
	scene = core.NewNode()
	ml := &ModelLoader{scene: scene}
	gui.Manager().Set(scene)
	windEnabled = false

	// Camera setup
	cam := camera.New(1)
	cam.SetPosition(0, 2, 2)
	cam.LookAt(&math32.Vector3{X: 0, Y: 1, Z: 0}, &math32.Vector3{X: 0, Y: 1, Z: 0})
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

	// Create surface
	surfaceGeom := geometry.NewPlane(20, 20)
	surfaceMat := material.NewStandard(math32.NewColor("Green"))
	surfaceMesh := graphic.NewMesh(surfaceGeom, surfaceMat)
	surfaceMesh.SetRotationX(-math32.Pi / 2)
	scene.Add(surfaceMesh)

	// Setup wind sources and UI
	windSources := initializeWindSources(scene)
	initializeUI(scene, windSources, ml)

	// Initialize fluid simulation
	initializeFluidSimulation(scene, windSources)

	// Lights and helpers
	scene.Add(light.NewAmbient(&math32.Color{R: 1.0, G: 1.0, B: 1.0}, 0.8))
	pointLight := light.NewPoint(&math32.Color{R: 1, G: 1, B: 1}, 5.0)
	pointLight.SetPosition(1, 0, 2)
	scene.Add(pointLight)
	scene.Add(helper.NewAxes(1.0))

	a.Gls().ClearColor(0.5, 0.5, 0.5, 1.0)

	// Application loop
	lastParticleTime := time.Now()
	a.Run(func(renderer *renderer.Renderer, deltaTime time.Duration) {
		a.Gls().Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)
		renderer.Render(scene, cam)

		log.Printf("Scene children count: %d, Wind particles: %d", len(scene.Children()), len(windParticles))

		// Continuous particle generation from wind sources
		if windEnabled {
			if time.Since(lastParticleTime).Milliseconds() >= 100 { // Spawn every 100ms
				for _, wind := range windSources {
					windParticles = append(windParticles, createWindParticle(wind.Position, wind.Direction))
					log.Printf("Spawning particle from wind source at: %v, Direction: %v", wind.Position, wind.Direction)
				}
				lastParticleTime = time.Now()
			}
		}

		if mesh != nil {
			log.Printf("Mesh is present at position: %v", mesh.Position())
			updatePhysics(mesh, windSources, float32(deltaTime.Seconds()))
		} else {
			log.Println("Mesh is nil")
		}
		updateWindParticles(float32(deltaTime.Seconds()), scene, mesh)

		// Simulate fluid dynamics
		simulateFluid(float32(deltaTime.Seconds()))
	})

	// Save simulation data
	saveSimulationData()
}
