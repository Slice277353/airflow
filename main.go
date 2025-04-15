package main

import (
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

// Import ModelLoader from model_loader.go

func main() {
	a := app.App()
	scene = core.NewNode()
	ml := &ModelLoader{scene: scene}
	gui.Manager().Set(scene)
	windEnabled = false

	// Camera setup
	cam := camera.New(1)
	cam.SetPosition(0, 2, 5)
	cam.LookAt(&math32.Vector3{X: 0, Y: 1, Z: 0}, &math32.Vector3{X: 0, Y: 0, Z: 1})
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
	initializeUI(scene, &windSources, ml, cam)


	// Initialize global fluid simulation with wind sources
	initializeFluidSimulation(scene, windSources)

	// Lights and helpers
	scene.Add(light.NewAmbient(&math32.Color{R: 1.0, G: 1.0, B: 1.0}, 0.8))
	pointLight := light.NewPoint(&math32.Color{R: 1, G: 1, B: 1}, 5.0)
	pointLight.SetPosition(1, 0, 2)
	scene.Add(pointLight)
	scene.Add(helper.NewAxes(1.0))

	a.Gls().ClearColor(0.5, 0.5, 0.5, 1.0)

	// Application loop
	a.Run(func(renderer *renderer.Renderer, deltaTime time.Duration) {
		a.Gls().Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)

		if windEnabled {
			// Use the existing simulateFluid function
			simulateFluid(float32(deltaTime.Seconds()))
		}

		renderer.Render(scene, cam)
	})
}
