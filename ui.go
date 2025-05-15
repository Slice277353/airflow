package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/g3n/engine/math32"
	"github.com/g3n/engine/texture"

	localcam "github.com/g3n/demos/hellog3n/camera"
	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/gui"
	"github.com/g3n/engine/window"
)

var (
	globalPlotsPanel *gui.Panel
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

	// Add analyze button (moved up before it's referenced)
	analyzeBtn := gui.NewButton("Start Recording")
	analyzeBtn.SetSize(120, 30)
	analyzeBtn.SetPosition(140, 10)
	analyzeBtn.SetEnabled(false) // Initially disabled
	controlPanel.Add(analyzeBtn)

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
			analyzeBtn.SetEnabled(true) // Enable recording while wind is running
			initializeFluidSimulation(scene, *windSources)
		} else {
			// Stopping wind - save data if we were recording
			if isRecording {
				stopRecording()

				// Save and process the data
				filepath, err := saveSimulationData()
				if err != nil {
					log.Println("Error saving simulation data:", err)
				} else {
					log.Printf("Saved simulation data to: %s", filepath)

					// Process with Python script using virtual environment
					cmd := exec.Command(getPythonPath(), "script.py", filepath)
					output, err := cmd.CombinedOutput()
					if err != nil {
						log.Printf("Error running analysis script: %v\nOutput: %s", err, string(output))
					} else {
						log.Printf("Analysis complete: %s", string(output))

						// Update plot display
						updatePlots(plotsPanel, filepath)
					}
				}
			}

			// Clean up simulation
			windEnabled = false
			btn.Label.SetText("Wind OFF")
			analyzeBtn.Label.SetText("Start Recording")
			analyzeBtn.SetEnabled(false)
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

	// Analyze (now Recording) button click handler
	analyzeBtn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		if !windEnabled {
			log.Println("Wind must be enabled to record simulation")
			return
		}

		if !isRecording {
			// Start recording
			startRecording()
			analyzeBtn.Label.SetText("Stop Recording")
		} else {
			// Stop recording but keep wind running
			stopRecording()
			analyzeBtn.Label.SetText("Start Recording")
		}
	})

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
		// Create a function to handle mouse click for placement
		mouseHandler := func(evname string, ev interface{}) {
			mev := ev.(*window.MouseEvent)

			// Create a ray from the camera through the mouse position
			width, _ := window.Get().GetSize()
			x := 2.0*float32(mev.Xpos)/float32(width) - 1.0
			y := -2.0*float32(mev.Ypos)/float32(width) + 1.0

			ray := localcam.NewRayFromMouse(cam, x, y)

			// Calculate intersection with y=1 plane (ground plane)
			groundNormal := &math32.Vector3{X: 0, Y: 1, Z: 0}
			groundPoint := &math32.Vector3{X: 0, Y: 1, Z: 0}

			// Get ray vectors
			rayOrigin := ray.Origin()
			rayDir := ray.Direction()

			// Calculate intersection using plane equation
			denom := rayDir.Dot(groundNormal)
			if math32.Abs(denom) > 1e-6 {
				p0l0 := groundPoint.Clone().Sub(&rayOrigin)
				t := p0l0.Dot(groundNormal) / denom
				if t >= 0 {
					// Calculate intersection point
					intersectPoint := rayOrigin.Clone()
					directionScaled := rayDir.Clone().MultiplyScalar(t)
					intersectPoint.Add(directionScaled)

					// Add the wind source at the intersection point
					*windSources = addWindSource(*windSources, scene, *intersectPoint)
					updateWindControls(controlPanel, windSources)
				}
			}

			// Remove the mouse handler after placement
			window.Get().UnsubscribeID(window.OnMouseDown, "wind_source_placement")
		}

		// Subscribe to mouse click with an ID for later removal
		window.Get().SubscribeID(window.OnMouseDown, "wind_source_placement", mouseHandler)
	})
	controlPanel.Add(addWindBtn)

	// Start recording button
	recordBtn := gui.NewButton("Start Recording")
	recordBtn.SetSize(120, 30)
	recordBtn.SetPosition(140, 50)
	recordBtn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		if len(simulationHistory) == 0 {
			recordBtn.Label.SetText("Stop Recording")
			simulationHistory = make([]SimulationSnapshot, 0)
			// Initialize first frame immediately
			recordSimulationFrame()
		} else {
			recordBtn.Label.SetText("Start Recording")
			// Save data when stopping
			if len(simulationHistory) > 0 {
				_, err := saveSimulationData()
				if err != nil {
					log.Println("Error saving simulation data:", err)
				}
				// Clear history after saving
				simulationHistory = nil
			}
		}
	})
	controlPanel.Add(recordBtn)

	// Export data button
	exportBtn := gui.NewButton("Export Data")
	exportBtn.SetSize(120, 30)
	exportBtn.SetPosition(140, 90)
	exportBtn.Subscribe(gui.OnClick, func(name string, ev interface{}) {
		if filepath, err := saveSimulationData(); err != nil {
			log.Println("Error saving data:", err)
		} else {
			// Run Python script with the exported data using virtual environment
			cmd := exec.Command(getPythonPath(), "script.py", filepath)
			if err := cmd.Run(); err != nil {
				log.Println("Error running analysis script:", err)
			}
		}
	})
	controlPanel.Add(exportBtn)

	updateWindControls(controlPanel, windSources)
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
		speedInput.Subscribe(gui.OnChange, func(name string, ev interface{}) {
			if val, err := strconv.ParseFloat(speedInput.Text(), 32); err == nil {
				(*windSources)[i].Speed = float32(val)
			}
		})
		panel.Add(speedInput)
		y += 25

		// Temperature control
		tempLabel := gui.NewLabel("Temp:")
		tempLabel.SetPosition(20, y)
		panel.Add(tempLabel)

		tempInput := gui.NewEdit(60, fmt.Sprintf("%.1f", (*windSources)[i].Temperature))
		tempInput.SetPosition(80, y)
		tempInput.Subscribe(gui.OnChange, func(name string, ev interface{}) {
			if val, err := strconv.ParseFloat(tempInput.Text(), 32); err == nil {
				(*windSources)[i].Temperature = float32(val)
			}
		})
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

		// Update direction handler
		updateDirFunc := func() {
			x, _ := strconv.ParseFloat(xInput.Text(), 32)
			y, _ := strconv.ParseFloat(yInput.Text(), 32)
			z, _ := strconv.ParseFloat(zInput.Text(), 32)
			dir := math32.NewVector3(float32(x), float32(y), float32(z))
			if dir.Length() > 0 {
				dir.Normalize()
				(*windSources)[i].Direction = *dir
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
}
