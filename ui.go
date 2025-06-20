package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/texture"

	localcam "github.com/g3n/demos/hellog3n/camera"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/window"
)

var (
	globalPlotsPanel *gui.Panel
	// --- Remade wind source system ---
	draggingWindSourceIdx = -1
	dragOffset            *math32.Vector3 // Offset between wind source and mouse at drag start
	windSourceControlMode = "mouse"       // "mouse" or "wasd"
	modeLabel             *gui.Label
)

// getPythonPath returns the appropriate Python interpreter path based on OS
func getPythonPath() string {
	if runtime.GOOS == "windows" {
		return ".venv/Scripts/python"
	}
	// Linux, macOS, and other Unix-like systems
	return ".venv/bin/python"
}

func initializeUI(panel *gui.Panel, windSources *[]WindSource, ml *ModelLoader, cam camera.ICamera) {
	// Create left control panel
	controlPanel = gui.NewPanel(300, 400)
	controlPanel.SetPosition(10, 10)
	controlPanel.SetColor4(&math32.Color4{R: 0.2, G: 0.2, B: 0.2, A: 0.8})
	scene.Add(controlPanel)

	// Create right panel for plots
	plotsPanel := gui.NewPanel(450, 900) // Wide enough for plots and padding
	plotsPanel.SetColor4(&math32.Color4{R: 0.2, G: 0.2, B: 0.2, A: 0.8})
	plotsPanel.SetVisible(false) // Initially hidden
	scene.Add(plotsPanel)

	// Store plotsPanel in a package-level variable so we can access it from anywhere
	globalPlotsPanel = plotsPanel

	// Position right panel (will be updated when window is resized)
	width, _ := window.Get().GetSize()
	plotsPanel.SetPosition(float32(width)-plotsPanel.Width()-10, 10)

	// Toggle wind button
	btn := gui.NewButton("Wind OFF")
	btn.SetPosition(10, 10)
	btn.SetSize(80, 30)

	// Wind button click handler
	btn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		if !windEnabled {
			// Starting wind
			windEnabled = true
			btn.Label.SetText("Wind ON")
			initializeFluidSimulation(scene, *windSources)
			startRecording() // Start recording simulation data
		} else {
			// Stopping wind - stop recording and process
			windEnabled = false
			btn.Label.SetText("Wind OFF")

			// --- Save simulation data and run Python script ---
			filepath, err := saveSimulationData()
			if err != nil {
				log.Printf("Error saving simulation data: %v", err)
				return
			}

			pythonPath := getPythonPath()
			cmd := exec.Command(pythonPath, "script.py", filepath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			log.Printf("Running Python script: %s script.py %s", pythonPath, filepath)
			err = cmd.Run()
			if err != nil {
				log.Printf("Error running Python script: %v", err)
				return
			}

			// Update plots panel with new images and info panel (forces)
			updatePlots(globalPlotsPanel, filepath)

			// Now clear wind and fluid particles
			for _, p := range windParticles {
				if p != nil && p.Mesh != nil {
					scene.Remove(p.Mesh)
				}
			}
			windParticles = nil
			clearFluidParticles(scene)
		}
	})
	controlPanel.Add(btn)

	// Import model button
	importBtn := gui.NewButton("Import Model")
	importBtn.SetSize(120, 30)
	importBtn.SetPosition(10, 50)
	importBtn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		filePath, err := openModelFileDialog()
		if err != nil || filePath == "" {
			log.Println("No file selected or error:", err)
			return
		}

		if mesh != nil {
			scene.Remove(mesh)
			mesh = nil
		}
		ml.models = nil

		if err := ml.LoadModel(filePath); err != nil {
			log.Println("Error loading model:", err)
			return
		}

		if len(ml.models) > 0 {
			mesh = ml.models[0]
			scene.Add(mesh)
			mesh.SetPosition(0, 1, 0)
		}
	})
	controlPanel.Add(importBtn)

	// Add wind source button with mouse placement
	addWindBtn := gui.NewButton("Add Wind Source")
	addWindBtn.SetSize(120, 30)
	addWindBtn.SetPosition(10, 90)
	addWindBtn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		const subID = "wind_source_placement"
		mouseHandler := func(evname string, ev interface{}) {
			mev := ev.(*window.MouseEvent)
			pt := getSceneIntersection(mev, cam, scene)
			if pt == nil {
				window.Get().UnsubscribeID(window.OnMouseDown, subID)
				return
			}
			x, z := clampToEnvironment(pt.X, pt.Z)
			pt.X = x
			pt.Z = z
			*windSources = addWindSourceClamped(*windSources, scene, *pt)
			updateWindControls(controlPanel, windSources)
			window.Get().UnsubscribeID(window.OnMouseDown, subID)
		}
		window.Get().SubscribeID(window.OnMouseDown, subID, mouseHandler)
	})
	controlPanel.Add(addWindBtn)

	updateWindControls(controlPanel, windSources)
	updateModeLabel()
	enableWindSourceWASDControl(windSources)
}

func updateWindControls(panel *gui.Panel, windSources *[]WindSource) {
	// Remove existing controls by getting rid of all children after index 3
	children := panel.Children()
	for i := len(children) - 1; i >= 4; i-- {
		if guiChild, ok := children[i].(gui.IPanel); ok {
			panel.Remove(guiChild)
		}
	}
	y := float32(130)

	// Add controls for each wind source
	for i := range *windSources {
		idx := i // capture the current value of i
		// Source label
		label := gui.NewLabel(fmt.Sprintf("Wind Source %d", i+1))
		label.SetPosition(10, y)
		panel.Add(label)
		y += 25

		// Speed control
		speedLabel := gui.NewLabel("Speed:")
		speedLabel.SetPosition(20, y)
		panel.Add(speedLabel)

		speedInput := gui.NewEdit(60, fmt.Sprintf("%.1f", (*windSources)[i].Speed))
		speedInput.SetPosition(80, y)
		panel.Add(speedInput)
		y += 25

		// Temperature control
		tempLabel := gui.NewLabel("Temp:")
		tempLabel.SetPosition(20, y)
		panel.Add(tempLabel)

		tempInput := gui.NewEdit(60, fmt.Sprintf("%.1f", (*windSources)[i].Temperature))
		tempInput.SetPosition(80, y)
		panel.Add(tempInput)
		y += 25

		// Direction controls
		dirLabel := gui.NewLabel("Direction:")
		dirLabel.SetPosition(20, y)
		panel.Add(dirLabel)
		y += 25

		// X direction
		xInput := gui.NewEdit(40, fmt.Sprintf("%.1f", (*windSources)[i].Direction.X))
		xInput.SetPosition(30, y)
		panel.Add(xInput)

		// Y direction
		yInput := gui.NewEdit(40, fmt.Sprintf("%.1f", (*windSources)[i].Direction.Y))
		yInput.SetPosition(80, y)
		panel.Add(yInput)

		// Z direction
		zInput := gui.NewEdit(40, fmt.Sprintf("%.1f", (*windSources)[i].Direction.Z))
		zInput.SetPosition(130, y)
		panel.Add(zInput)

		// Update speed control handler
		speedInput.Subscribe(gui.OnChange, func(name string, ev interface{}) {
			if val, err := strconv.ParseFloat(speedInput.Text(), 32); err == nil {
				(*windSources)[idx].Speed = float32(val)
				// Force immediate update of the vector field
				updateVectorFieldFromSource(&(*windSources)[idx])
			}
		})

		// Update temperature control handler
		tempInput.Subscribe(gui.OnChange, func(name string, ev interface{}) {
			if val, err := strconv.ParseFloat(tempInput.Text(), 32); err == nil {
				(*windSources)[idx].Temperature = float32(val)
				// Force immediate update of the vector field
				updateVectorFieldFromSource(&(*windSources)[idx])
			}
		})

		// Update direction handler with immediate effect
		updateDirFunc := func() {
			x, _ := strconv.ParseFloat(xInput.Text(), 32)
			y, _ := strconv.ParseFloat(yInput.Text(), 32)
			z, _ := strconv.ParseFloat(zInput.Text(), 32)
			dir := math32.NewVector3(float32(x), float32(y), float32(z))
			if dir.Length() > 0 {
				dir.Normalize()
				(*windSources)[idx].Direction = *dir
				// Force immediate update of the vector field
				updateVectorFieldFromSource(&(*windSources)[idx])
			}
		}
		xInput.Subscribe(gui.OnChange, func(name string, ev interface{}) { updateDirFunc() })
		yInput.Subscribe(gui.OnChange, func(name string, ev interface{}) { updateDirFunc() })
		zInput.Subscribe(gui.OnChange, func(name string, ev interface{}) { updateDirFunc() })

		y += 40
	}
}

func createNumericInput(defaultValue float32, x, y float32, onChange func(value float32)) *gui.Edit {
	textInput := gui.NewEdit(100, fmt.Sprintf("%.2f", defaultValue))
	textInput.SetPosition(x, y)

	textInput.Subscribe(gui.OnChange, func(name string, ev interface{}) {
		text := textInput.Text()
		if value, err := strconv.ParseFloat(text, 32); err == nil {
			onChange(float32(value))
		}
	})

	return textInput
}

// Add this helper function to update plots
func updatePlots(plotsPanel *gui.Panel, filepath string) {
	// Define plot files
	basePath := filepath[:len(filepath)-5]
	plotFiles := map[string]string{
		"velocity":   basePath + "_velocity.png",
		"magnitude":  basePath + "_magnitude.png",
		"trajectory": basePath + "_trajectory.png",
		"position":   basePath + "_position.png",
	}

	// Check if all plot files exist
	for plotType, plotPath := range plotFiles {
		if _, err := os.Stat(plotPath); os.IsNotExist(err) {
			log.Printf("%s plot file was not created at expected path: %s", plotType, plotPath)
			return
		}
	}
	log.Println("All plot files created successfully")

	// Clear existing plots
	for _, child := range plotsPanel.Children() {
		if panel, ok := child.(gui.IPanel); ok {
			plotsPanel.Remove(panel)
		}
	}

	// Create and position plots
	plotWidth := float32(400)
	plotHeight := float32(200)
	padding := float32(10)

	// Define the order of plots (top to bottom)
	plotOrder := []string{"velocity", "magnitude", "trajectory", "position"}

	// Create and position each plot in order
	for i, plotType := range plotOrder {
		plotPath := plotFiles[plotType]

		// Create container panel for each plot
		container := gui.NewPanel(plotWidth, plotHeight)
		container.SetPosition(padding, padding+float32(i)*(plotHeight+padding))
		plotsPanel.Add(container)

		// Create texture from image
		tex, err := texture.NewTexture2DFromImage(plotPath)
		if err != nil {
			log.Printf("Error loading texture for %s: %v", plotType, err)
			continue
		}

		// Create image with texture
		img, err := gui.NewImage(plotPath)
		if err != nil || img == nil {
			log.Printf("Error creating image panel for %s: %v", plotType, err)
			continue
		}

		img.SetTexture(tex)
		img.SetSize(plotWidth, plotHeight)
		container.Add(img)

		// Add click handler for the container
		container.Subscribe(gui.OnMouseDown, func(name string, ev interface{}) {
			// Create full-screen overlay
			width, height := window.Get().GetSize()
			overlay := gui.NewPanel(float32(width), float32(height))
			overlay.SetColor4(&math32.Color4{R: 0, G: 0, B: 0, A: 0.9})
			overlay.SetPosition(0, 0)
			scene.Add(overlay)

			// Create larger version of the plot
			largeImg, _ := gui.NewImage(plotPath)
			largeTex, _ := texture.NewTexture2DFromImage(plotPath)
			largeImg.SetTexture(largeTex)

			// Calculate size to maintain aspect ratio
			imgAspect := plotWidth / plotHeight
			screenAspect := float32(width) / float32(height)
			var imgWidth, imgHeight float32

			if imgAspect > screenAspect {
				imgWidth = float32(width) * 0.9
				imgHeight = imgWidth / imgAspect
			} else {
				imgHeight = float32(height) * 0.9
				imgWidth = imgHeight * imgAspect
			}

			largeImg.SetSize(imgWidth, imgHeight)
			largeImg.SetPosition((float32(width)-imgWidth)/2, (float32(height)-imgHeight)/2)
			overlay.Add(largeImg)

			// Add close button
			closeBtn := gui.NewButton("×")
			closeBtn.SetSize(40, 40)
			closeBtn.SetPosition(float32(width)-50, 10)
			closeBtn.Label.SetFontSize(24)
			closeBtn.Label.SetColor(&math32.Color{R: 1, G: 1, B: 1})
			closeBtn.SetColor(&math32.Color{R: 0.5, G: 0, B: 0})
			overlay.Add(closeBtn)

			closeBtn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
				scene.Remove(overlay)
			})

			// Track whether we're clicking on the image
			var clickedOnImage bool

			largeImg.Subscribe(gui.OnMouseDown, func(name string, ev interface{}) {
				clickedOnImage = true
			})

			largeImg.Subscribe(gui.OnMouseUp, func(name string, ev interface{}) {
				clickedOnImage = false
			})

			// Close on background click
			overlay.Subscribe(gui.OnMouseDown, func(name string, ev interface{}) {
				if !clickedOnImage && ev.(*window.MouseEvent).Button == window.MouseButtonLeft {
					scene.Remove(overlay)
				}
			})
		})

		// Add hover effect
		container.Subscribe(gui.OnCursor, func(name string, ev interface{}) {
			container.SetColor4(&math32.Color4{R: 1, G: 1, B: 1, A: 0.1})
		})
		container.Subscribe(gui.OnCursorLeave, func(name string, ev interface{}) {
			container.SetColor4(&math32.Color4{R: 0, G: 0, B: 0, A: 0})
		})

		log.Printf("Added %s plot at position (%.0f, %.0f)",
			plotType, padding, padding+float32(i)*(plotHeight+padding))
	}

	// Show the plots panel
	plotsPanel.SetVisible(true)

	// --- Add info panel for lift and drag forces ---
	// Remove any existing info panel
	for _, child := range scene.Children() {
		if panel, ok := child.(*gui.Panel); ok && panel.Name() == "info_panel" {
			scene.Remove(panel)
		}
	}

	// Calculate forces (call Go functions)
	avgDrag := calculateAverageDragForce()
	avgLift := calculateAverageLiftForce()

	// Create info panel
	_, winHeight := window.Get().GetSize()
	infoPanel := gui.NewPanel(260, 80)
	infoPanel.SetName("info_panel")
	infoPanel.SetColor4(&math32.Color4{R: 0.1, G: 0.1, B: 0.1, A: 0.85})
	infoPanel.SetBorders(1, 1, 1, 1)
	infoPanel.SetPosition(10, float32(winHeight)-90) // Lower left corner

	// Add labels
	labelTitle := gui.NewLabel("Simulation Forces")
	labelTitle.SetFontSize(18)
	labelTitle.SetColor(&math32.Color{R: 1, G: 1, B: 1})
	labelTitle.SetPosition(10, 8)
	infoPanel.Add(labelTitle)

	labelDrag := gui.NewLabel(fmt.Sprintf("Average Drag: %.3f N", avgDrag))
	labelDrag.SetFontSize(15)
	labelDrag.SetColor(&math32.Color{R: 0.8, G: 0.8, B: 1})
	labelDrag.SetPosition(10, 32)
	infoPanel.Add(labelDrag)

	labelLift := gui.NewLabel(fmt.Sprintf("Average Lift: %.3f N", avgLift))
	labelLift.SetFontSize(15)
	labelLift.SetColor(&math32.Color{R: 0.8, G: 1, B: 0.8})
	labelLift.SetPosition(10, 54)
	infoPanel.Add(labelLift)

	scene.Add(infoPanel)
}

// --- Remade wind source system ---
// Clamp a position to environment bounds
func clampToEnvironment(x, z float32) (float32, float32) {
	if x < -10 {
		x = -10
	}
	if x > 10 {
		x = 10
	}
	if z < -10 {
		z = -10
	}
	if z > 10 {
		z = 10
	}
	return x, z
}

// Add a wind source at a given position (clamped if needed)
func addWindSourceClamped(sources []WindSource, scene *core.Node, position math32.Vector3) []WindSource {
	x, z := clampToEnvironment(position.X, position.Z)
	position.X = x
	position.Z = z
	return addWindSource(sources, scene, position)
}

// Helper: get intersection with the first visible mesh (e.g., floor or imported model)
func getSceneIntersection(mev *window.MouseEvent, cam camera.ICamera, scene *core.Node) *math32.Vector3 {
	width, height := window.Get().GetSize()
	xn := 2.0*float32(mev.Xpos)/float32(width) - 1.0
	yn := -2.0*float32(mev.Ypos)/float32(height) + 1.0
	ray := localcam.NewRayFromMouse(cam, xn, yn)

	// Try to intersect with the first mesh in the scene (e.g., the floor)
	for _, child := range scene.Children() {
		if mesh, ok := child.(*graphic.Mesh); ok {
			if pt, ok := rayMeshIntersection(ray, mesh); ok {
				return pt
			}
		}
	}
	// Fallback: intersect with y=0 plane
	groundNormal := math32.NewVector3(0, 1, 0)
	groundPoint := math32.NewVector3(0, 0, 0)
	rayOrigin := ray.Origin()
	rayDir := ray.Direction()
	denom := rayDir.Dot(groundNormal)
	if math32.Abs(denom) > 1e-6 {
		p0l0 := groundPoint.Clone().Sub(&rayOrigin)
		t := p0l0.Dot(groundNormal) / denom
		if t >= 0 {
			intersectPoint := (&rayOrigin).Clone().Add((&rayDir).Clone().MultiplyScalar(t))
			return intersectPoint
		}
	}
	return nil
}

// Helper: ray-mesh intersection (only works for planes and simple meshes)
func rayMeshIntersection(ray *math32.Ray, mesh *graphic.Mesh) (*math32.Vector3, bool) {
	geom := mesh.GetGeometry()
	if geom == nil {
		return nil, false
	}
	posAttr := geom.VBO(0) // 0 = position
	if posAttr == nil {
		return nil, false
	}
	positions := posAttr.Buffer().ToFloat32()
	indices := geom.Indices()
	worldMatrix := mesh.ModelMatrix()
	if len(indices) == 0 {
		for i := 0; i+2 < len(positions)/3; i += 3 {
			a := math32.NewVector3(positions[3*i+0], positions[3*i+1], positions[3*i+2]).ApplyMatrix4(worldMatrix)
			b := math32.NewVector3(positions[3*(i+1)+0], positions[3*(i+1)+1], positions[3*(i+1)+2]).ApplyMatrix4(worldMatrix)
			c := math32.NewVector3(positions[3*(i+2)+0], positions[3*(i+2)+1], positions[3*(i+2)+2]).ApplyMatrix4(worldMatrix)
			if pt, ok := rayTriangleIntersection(ray, *a, *b, *c); ok {
				return pt, true
			}
		}
	} else {
		for i := 0; i+2 < len(indices); i += 3 {
			ia := indices[i]
			ib := indices[i+1]
			ic := indices[i+2]
			a := math32.NewVector3(positions[3*ia+0], positions[3*ia+1], positions[3*ia+2]).ApplyMatrix4(worldMatrix)
			b := math32.NewVector3(positions[3*ib+0], positions[3*ib+1], positions[3*ib+2]).ApplyMatrix4(worldMatrix)
			c := math32.NewVector3(positions[3*ic+0], positions[3*ic+1], positions[3*ic+2]).ApplyMatrix4(worldMatrix)
			if pt, ok := rayTriangleIntersection(ray, *a, *b, *c); ok {
				return pt, true
			}
		}
	}
	return nil, false
}

// Helper: ray-triangle intersection (Möller–Trumbore algorithm)
func rayTriangleIntersection(ray *math32.Ray, a, b, c math32.Vector3) (*math32.Vector3, bool) {
	e1 := (&b).Clone().Sub(&a)
	e2 := (&c).Clone().Sub(&a)
	dir := ray.Direction()
	h := (&dir).Clone().Cross(e2)
	det := e1.Dot(h)
	if det > -1e-6 && det < 1e-6 {
		return nil, false
	}
	invDet := 1.0 / det
	org := ray.Origin()
	s := (&org).Clone().Sub(&a)
	u := s.Dot(h) * invDet
	if u < 0.0 || u > 1.0 {
		return nil, false
	}
	q := s.Clone().Cross(e1)
	v := dir.Dot(q) * invDet
	if v < 0.0 || u+v > 1.0 {
		return nil, false
	}
	t := e2.Dot(q) * invDet
	if t < 0 {
		return nil, false
	}
	intersect := (&org).Clone().Add((&dir).Clone().MultiplyScalar(t))
	return intersect, true
}

// --- Mode indicator label ---
func updateModeLabel() {
	if modeLabel == nil {
		modeLabel = gui.NewLabel("")
		modeLabel.SetFontSize(32)
		modeLabel.SetColor(&math32.Color{R: 1, G: 1, B: 0})
		width, height := window.Get().GetSize()
		modeLabel.SetPosition(float32(width)/2-60, float32(height)/2-30)
		scene.Add(modeLabel)
	}
	modeLabel.SetVisible(true)
}

// --- WASD control logic ---
func enableWindSourceWASDControl(windSources *[]WindSource) {
	const moveStep = 0.2
	window.Get().SubscribeID(window.OnKeyDown, "wasd_mode_keydown", func(evname string, ev interface{}) {
		kev := ev.(*window.KeyEvent)
		if windSourceControlMode != "wasd" || draggingWindSourceIdx < 0 {
			return
		}
		ws := &(*windSources)[draggingWindSourceIdx]
		switch kev.Key {
		case window.KeyW:
			ws.Position.Z -= moveStep
		case window.KeyS:
			ws.Position.Z += moveStep
		case window.KeyA:
			ws.Position.X -= moveStep
		case window.KeyD:
			ws.Position.X += moveStep
		}
		x, z := clampToEnvironment(ws.Position.X, ws.Position.Z)
		ws.Position.X = x
		ws.Position.Z = z
		if ws.Node != nil {
			ws.Node.SetPositionVec(&ws.Position)
		}
		updateVectorFieldFromSource(ws)
		updateWindControls(controlPanel, windSources)
	})
	window.Get().SubscribeID(window.OnKeyDown, "wasd_mode_esc", func(evname string, ev interface{}) {
		kev := ev.(*window.KeyEvent)
		if kev.Key == window.KeyEscape && windSourceControlMode == "wasd" {
			windSourceControlMode = "mouse"
			updateModeLabel()
		}
	})
}
