package main

import (
	"fmt"
	"github.com/g3n/engine/app"
	"github.com/g3n/engine/math32"
	"log"
	"strconv"
	"strings"

	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/window"
)

func initializeUI(scene *core.Node, windSources []WindSource, ml *ModelLoader, cam camera.ICamera) {
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

	emptyBtn := gui.NewButton("Import an object")
	emptyBtn.SetSize(120, 40)
	scene.Add(emptyBtn)

	addWindBtn := gui.NewButton("Add Wind Source")
	addWindBtn.SetSize(120, 40)
	scene.Add(addWindBtn)

	waitingForWindPlacement := false

	updateButtonLayout := func(w, h int) {
		const minWidth, minHeight = 400, 200
		if w < minWidth || h < minHeight {
			emptyBtn.SetVisible(false)
			addWindBtn.SetVisible(false)
			return
		}
		emptyBtn.SetVisible(true)
		addWindBtn.SetVisible(true)

		btnWidth := float32(w) * 0.15
		btnHeight := float32(h) * 0.05
		btnX := float32(w) - btnWidth - float32(w)*0.05
		btnY := float32(h) * 0.1

		emptyBtn.SetSize(btnWidth, btnHeight)
		emptyBtn.SetPosition(btnX, btnY)

		addWindBtn.SetSize(btnWidth, btnHeight)
		addWindBtn.SetPosition(btnX, btnY+btnHeight+10)
	}

	app.App().Subscribe(window.OnWindowSize, func(evname string, ev interface{}) {
		w, h := app.App().GetSize()
		updateButtonLayout(w, h)
	})

	w, h := app.App().GetSize()
	updateButtonLayout(w, h)

	emptyBtn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		filePath, err := openFileDialog()
		if err != nil || filePath == "" {
			log.Println("No file selected or error:", err)
			return
		}

		log.Println("Selected file:", filePath)

		// Remove old model
		if mesh != nil {
			scene.Remove(mesh)
			mesh = nil
		}
		ml.models = nil

		// Load new model
		if err := ml.LoadModel(filePath); err != nil {
			log.Println("Error loading model:", err)
			return
		}

		if len(ml.models) > 0 {
			mesh = ml.models[0]
			scene.Add(mesh)
			// Set position directly (remove centering logic for now)
			mesh.SetPosition(0, 1, 0)
			log.Printf("New mesh loaded and added to scene at position: %v", mesh.Position())
		} else {
			log.Println("No models were loaded.")
			mesh = nil
		}
	})

	addWindBtn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		//defaultPos := *math32.NewVector3(0, 1, 0)
		//windSources = addWindSource(windSources, scene, defaultPos)
		//
		//newIndex := len(windSources) - 1
		//windSpeedInput := createNumericInput((windSources)[newIndex].Speed, 100, 200+float32(newIndex*50), func(value float32) {
		//	(windSources)[newIndex].Speed = value
		//})
		//scene.Add(windSpeedInput)
		waitingForWindPlacement = true
		log.Println("Click on the scene to place the wind source")
	})
	app.App().Subscribe(window.OnMouseDown, func(evname string, ev interface{}) {
		if !waitingForWindPlacement {
			return
		}

		mev := ev.(*window.MouseEvent)
		if mev.Button != window.MouseButtonLeft {
			return
		}

		// Get the mouse position in normalized device coordinates
		w, h := app.App().GetSize()
		x := float32(mev.Xpos)/float32(w)*2 - 1
		y := -(float32(mev.Ypos)/float32(h)*2 - 1)

		// Get the projection and view matrices
		projMatrix := &math32.Matrix4{}
		viewMatrix := &math32.Matrix4{}
		cam.ProjMatrix(projMatrix)
		cam.ViewMatrix(viewMatrix)

		// Compute the combined view-projection matrix
		viewProjMatrix := &math32.Matrix4{}
		viewProjMatrix.MultiplyMatrices(projMatrix, viewMatrix)

		// Compute the inverse of the view-projection matrix
		invViewProjMatrix := &math32.Matrix4{}
		err := invViewProjMatrix.GetInverse(viewProjMatrix)
		if err != nil {
			log.Println("failed to invert view-projection matrix")
			return
		}

		// Define near and far points in NDC
		nearNDC := math32.NewVector4(x, y, 0, 1) // Near plane (z=0 in NDC)
		farNDC := math32.NewVector4(x, y, 1, 1)  // Far plane (z=1 in NDC)

		nearWorld := &math32.Vector4{}
		farWorld := &math32.Vector4{}
		nearNDC.ApplyMatrix4(invViewProjMatrix)
		farNDC.ApplyMatrix4(invViewProjMatrix)
		nearWorld.Copy(nearNDC)
		farWorld.Copy(farNDC)

		// Perspective divide to convert from homogeneous coordinates to 3
		// Perspective divide to convert from homogeneous coordinates to 3D
		near := &math32.Vector3{}
		far := &math32.Vector3{}
		if nearWorld.W != 0 {
			near.X = nearWorld.X / nearWorld.W
			near.Y = nearWorld.Y / nearWorld.W
			near.Z = nearWorld.Z / nearWorld.W
		}
		if farWorld.W != 0 {
			far.X = farWorld.X / farWorld.W
			far.Y = farWorld.Y / farWorld.W
			far.Z = farWorld.Z / farWorld.W
		}

		// Compute the ray direction from near to far
		direction := far.Sub(near).Normalize()

		// Compute intersection with the ground plane (y=0)
		origin := cam.(*camera.Camera).GetNode().Position()
		t := -origin.Y / direction.Y // Solve for t where y=0: origin.Y + t*direction.Y = 0
		if t < 0 {
			log.Println("No intersection with ground plane")
			return
		}

		// Compute the intersection point
		intersectPoint := &math32.Vector3{}
		intersectPoint.X = origin.X + t*direction.X
		intersectPoint.Y = 0 // Ground plane
		intersectPoint.Z = origin.Z + t*direction.Z

		// Spawn the wind source at the intersected point
		addWindSource(windSources, scene, *intersectPoint)

		newIndex := len(windSources) - 1
		windSpeedInput := createNumericInput((windSources)[newIndex].Speed, 100, 200+float32(newIndex*50), func(value float32) {
			(windSources)[newIndex].Speed = value
		})
		scene.Add(windSpeedInput)

		log.Printf("Wind source added at position: %v", intersectPoint)
		waitingForWindPlacement = false
	})

	// Use global mass and dragCoefficient from physics.go
	massInput := createNumericInput(mass, 100, 100, func(value float32) {
		mass = value
	})
	scene.Add(massInput)

	dragInput := createNumericInput(dragCoefficient, 100, 150, func(value float32) {
		dragCoefficient = value
	})
	scene.Add(dragInput)

	for i, wind := range windSources {
		windSpeedInput := createNumericInput(wind.Speed, 100, 200+float32(i*50), func(value float32) {
			windSources[i].Speed = value
		})
		scene.Add(windSpeedInput)
	}
}

func createNumericInput(defaultValue float32, x, y float32, onChange func(value float32)) *gui.Edit {
	textInput := gui.NewEdit(100, fmt.Sprintf("%.2f", defaultValue))
	textInput.SetPosition(x, y)

	textInput.Subscribe(gui.OnChange, func(name string, ev interface{}) {
		text := textInput.Text()
		filteredText := filterNumericInput(text)
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
				textInput.SetText(fmt.Sprintf("%.2f", defaultValue))
			}
		}
	})

	return textInput
}

func filterNumericInput(input string) string {
	var builder strings.Builder
	dotCount := 0

	for i, char := range input {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		} else if char == '.' && dotCount == 0 {
			builder.WriteRune(char)
			dotCount++
		} else if char == '-' && i == 0 {
			builder.WriteRune(char)
		}
	}

	return builder.String()
}
