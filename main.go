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

var (
	scene             *core.Node
	mesh              *core.Node
	windEnabled       bool
	welcomeScreen     *gui.Panel
	controlPanel      *gui.Panel
	simulationStarted bool
	titleLabel        *gui.Label
	startButton       *gui.Button
)

// Import ModelLoader from model_loader.go

func setupWelcomeScreen(scene *core.Node) *gui.Panel {
	// Get window size
	width, height := window.Get().GetSize()

	// Create main background panel
	panel := gui.NewPanel(float32(width), float32(height))
	panel.SetColor4(&math32.Color4{R: 0.1, G: 0.1, B: 0.1, A: 0.95}) // Darker, more opaque base
	panel.SetPosition(0, 0)

	// Create multiple blur layers for a more pronounced blur effect
	for i := 0; i < 3; i++ {
		blurLayer := gui.NewPanel(float32(width), float32(height))
		blurLayer.SetColor4(&math32.Color4{R: 1, G: 1, B: 1, A: 0.05})
		blurLayer.SetPosition(float32(i)*0.5, float32(i)*0.5) // Slight offset for each layer
		panel.Add(blurLayer)
	}

	// Create content container panel with wider initial width
	contentPanel := gui.NewPanel(500, float32(height)*0.4)         // Match the width from updateWelcomeScreenLayout
	contentPanel.SetColor4(&math32.Color4{R: 0, G: 0, B: 0, A: 0}) // Transparent
	panel.Add(contentPanel)

	// Create title
	titleLabel = gui.NewLabel("Airflow Simulation")
	titleLabel.SetFontSize(48)
	titleLabel.SetColor(&math32.Color{R: 1, G: 1, B: 1})
	contentPanel.Add(titleLabel)

	// Create start button
	startButton = gui.NewButton("Start Simulation")
	startButton.Label.SetFontSize(24)
	contentPanel.Add(startButton)

	// Initial positioning
	updateWelcomeScreenLayout(width, height)

	startButton.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		simulationStarted = true
		scene.Remove(panel)
		controlPanel.SetVisible(true)
	})

	scene.Add(panel)
	return panel
}

func updateWelcomeScreenLayout(width, height int) {
	if titleLabel == nil || startButton == nil {
		return
	}

	// Calculate content container size and position
	containerWidth := float32(500)
	containerHeight := float32(height) * 0.4
	containerX := float32(width)/2 - containerWidth/2
	containerY := float32(height)/2 - containerHeight/2

	// Position content container
	if titleLabel.Parent() != nil {
		if contentPanel, ok := titleLabel.Parent().(*gui.Panel); ok {
			contentPanel.SetPosition(containerX, containerY)
			contentPanel.SetSize(containerWidth, containerHeight)
		}
	}

	// Calculate title dimensions more accurately - adjusted multiplier from 0.55 to 0.35
	titleWidth := float32(float64(len("Airflow Simulation")) * titleLabel.FontSize() * 0.45)

	// Center title horizontally and vertically
	titleX := containerWidth/2 - titleWidth/2
	titleY := containerHeight * 0.35 // Adjusted to 35% from top for better visual center

	titleLabel.SetPosition(titleX, titleY)

	// Make button size relative to container
	buttonWidth := math32.Min(containerWidth*0.4, 200)
	buttonHeight := buttonWidth * 0.3
	startButton.SetSize(buttonWidth, buttonHeight)

	// Position button below title with proper spacing
	buttonX := containerWidth/2 - buttonWidth/2
	buttonY := containerHeight * 0.65 // Position at 65% of container height
	startButton.SetPosition(buttonX, buttonY)

	// Update blur layers
	if welcomeScreen != nil {
		welcomeScreen.SetSize(float32(width), float32(height))
		for _, child := range welcomeScreen.Children() {
			if panel, ok := child.(*gui.Panel); ok {
				panel.SetSize(float32(width), float32(height))
			}
		}
	}
}

func main() {
	a := app.App()
	scene = core.NewNode()
	ml := &ModelLoader{scene: scene}
	gui.Manager().Set(scene)
	windEnabled = false
	simulationStarted = false

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
		if welcomeScreen != nil {
			welcomeScreen.SetSize(float32(width), float32(height))
			updateWelcomeScreenLayout(width, height)
		}
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
	controlPanel = gui.NewPanel(300, 400)
	controlPanel.SetPosition(10, 10)
	controlPanel.SetColor4(&math32.Color4{R: 0.2, G: 0.2, B: 0.2, A: 0.8})
	controlPanel.SetVisible(false)
	scene.Add(controlPanel)
	initializeUI(controlPanel, &windSources, ml, cam)

	// Create welcome screen
	welcomeScreen = setupWelcomeScreen(scene)

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

		if simulationStarted && windEnabled {
			simulateFluid(float32(deltaTime.Seconds()), scene)
		}

		renderer.Render(scene, cam)
	})
}
