package main

import (
	"log"
	"math/rand"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/graphic"
	"github.com/g3n/engine/material"
	"github.com/g3n/engine/math32"
)

type WindSource struct {
	Position    math32.Vector3
	Radius      float32
	Speed       float32
	Direction   math32.Vector3
	Node        *graphic.Mesh
	Spread      float32
	Temperature float32
}

type WindParticle struct {
	Mesh        *graphic.Mesh
	Velocity    math32.Vector3
	Position    *math32.Vector3
	Mass        float32
	Lifespan    float32
	Elapsed     float32
	Alive       bool
	Temperature float32
	Turbulence  float32
}

var windParticles []*WindParticle

var (
	windSources []WindSource
)

func initializeWindSources(scn *core.Node) []WindSource {
	scene = scn // Store scene globally
	windSources = []WindSource{
		{Position: *math32.NewVector3(5, 2, 5), Radius: 3.0, Speed: 8.0, Direction: *math32.NewVector3(-1, 0, -1).Normalize(), Spread: 0.2, Temperature: 25.0}, // Diagonal wind
		{Position: *math32.NewVector3(-5, 2, -5), Radius: 2.0, Speed: 6.0, Direction: *math32.NewVector3(1, 0, 1).Normalize(), Spread: 0.3, Temperature: 20.0}, // Opposite diagonal
	}

	for i := range windSources {

		sphereGeom := geometry.NewSphere(0.2, 16, 16)
		sphereMat := material.NewStandard(math32.NewColor("Red"))
		sphereMesh := graphic.NewMesh(sphereGeom, sphereMat)
		sphereMesh.SetPositionVec(&windSources[i].Position)
		windSources[i].Node = sphereMesh // Store the mesh in the WindSource struct
		scene.Add(sphereMesh)
	}

	return windSources
}

func addWindSource(sources []WindSource, scn *core.Node, position math32.Vector3) []WindSource {
	newWind := WindSource{
		Position:    position,
		Radius:      2.0,
		Speed:       5.0,
		Direction:   *math32.NewVector3(1, 0, 0).Normalize(),
		Spread:      0.2,
		Temperature: 22.0,
	}

	sphereGeom := geometry.NewSphere(0.2, 16, 16)
	sphereMat := material.NewStandard(math32.NewColor("Red"))
	sphereMesh := graphic.NewMesh(sphereGeom, sphereMat)
	sphereMesh.SetPositionVec(&newWind.Position)
	newWind.Node = sphereMesh
	scene.Add(sphereMesh)

	windSources = append(sources, newWind) // Update global windSources
	return windSources
}

func createWindParticle(scene *core.Node, position, direction math32.Vector3, speed float32) *WindParticle {
	// Add random spread to the direction
	spreadFactor := float32(0.2)
	randomDir := math32.NewVector3(
		(rand.Float32()-0.5)*spreadFactor,
		(rand.Float32()-0.5)*spreadFactor,
		(rand.Float32()-0.5)*spreadFactor,
	)
	newDir := *direction.Clone().Add(randomDir).Normalize()

	// Create visual representation
	particleGeom := geometry.NewCylinder(0.05, 0.2, 8, 1, true, true) // Smaller, more arrow-like particles
	particleMat := material.NewStandard(math32.NewColor("Cyan"))
	particleMat.SetOpacity(0.7)
	particleMat.SetTransparent(true)
	particleMesh := graphic.NewMesh(particleGeom, particleMat)

	// Set initial position
	particleMesh.SetPositionVec(&position)

	// Orient particle in direction of movement
	yaw := math32.Atan2(newDir.Z, newDir.X)
	pitch := math32.Asin(newDir.Y)
	particleMesh.SetRotation(pitch, yaw, 0)

	scene.Add(particleMesh)

	return &WindParticle{
		Mesh:        particleMesh,
		Velocity:    *newDir.MultiplyScalar(speed),
		Position:    position.Clone(),
		Mass:        1.0,
		Lifespan:    5.0,
		Elapsed:     0,
		Alive:       true,
		Temperature: 20.0,
		Turbulence:  rand.Float32() * 0.3,
	}
}

func updateWindParticles(deltaTime float32, scene *core.Node, mesh *core.Node) {
	var activeParticles []*WindParticle

	for _, particle := range windParticles {
		if particle == nil || !particle.Alive {
			continue
		}

		particle.Elapsed += deltaTime
		if particle.Elapsed >= particle.Lifespan {
			scene.Remove(particle.Mesh)
			continue
		}

		// Update physics
		updatePhysics(particle, mesh, deltaTime)

		// Remove particles that are too far from the scene
		if particle.Position.Length() > 20 {
			scene.Remove(particle.Mesh)
			continue
		}

		// Keep active particles
		activeParticles = append(activeParticles, particle)
	}

	windParticles = activeParticles
}

func calculateWindInfluence(particle *WindParticle, source *WindSource) (math32.Vector3, float32) {
	particlePos := math32.NewVector3(particle.Position.X, particle.Position.Y, particle.Position.Z)
	toParticle := *particlePos.Sub(&source.Position)
	distance := toParticle.Length()

	if distance > source.Radius {
		return *math32.NewVector3(0, 0, 0), 0
	}

	// Calculate influence based on distance (closer = stronger)
	influence := 1.0 - (distance / source.Radius)

	// Temperature affects vertical movement
	tempDiff := source.Temperature - particle.Temperature
	verticalForce := tempDiff * 0.01 // Adjust this multiplier as needed

	// Combine direction with temperature-based vertical movement
	windDir := source.Direction.Clone()
	windDir.Y += verticalForce
	windDir.Normalize()

	// Scale by source speed and influence
	return *windDir.MultiplyScalar(source.Speed * influence), influence
}

// Removed duplicate updatePhysics function to resolve redeclaration error.
// The implementation is assumed to exist in physics.go.

type VectorField struct {
	Width      int
	Height     int
	Depth      int
	AreaWidth  int
	AreaHeight int
	AreaDepth  int
	Field      [][][]Vector // 3D grid of vectors
}

type Vector struct {
	VX  float32
	VY  float32
	VZ  float32
	VX_ float32
	VY_ float32
	VZ_ float32
}

type Particle struct {
	X     float32
	Y     float32
	Z     float32
	OX    float32
	OY    float32
	OZ    float32
	VX    float32
	VY    float32
	VZ    float32
	Speed float32
	Mesh  *graphic.Mesh
}

var fluidParticles []Particle
var vectorField VectorField

func clamp(value, min, max float32) float32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func calcMagnitude3D(x, y, z float32) float32 {
	return float32(math32.Sqrt(x*x + y*y + z*z))
}

func initVectorField(width, height, depth, areaWidth, areaHeight, areaDepth int) VectorField {
	field := make([][][]Vector, areaWidth)
	for x := 0; x < areaWidth; x++ {
		field[x] = make([][]Vector, areaHeight)
		for y := 0; y < areaHeight; y++ {
			field[x][y] = make([]Vector, areaDepth)
			for z := 0; z < areaDepth; z++ {
				field[x][y][z] = Vector{VX: 0, VY: 0, VZ: -5, VX_: 0, VY_: 0, VZ_: 0}
			}
		}
	}
	return VectorField{
		Width:      width,
		Height:     height,
		Depth:      depth,
		AreaWidth:  areaWidth,
		AreaHeight: areaHeight,
		AreaDepth:  areaDepth,
		Field:      field,
	}
}

func initParticles(count int, scene *core.Node) []Particle {
	particles := make([]Particle, count)

	// Calculate grid dimensions for even distribution
	gridSize := int(math32.Pow(float32(count), 1.0/3.0))
	spacing := float32(20.0) / float32(gridSize)

	index := 0
	// Create an evenly spaced grid of particles
	for x := 0; x < gridSize && index < count; x++ {
		for y := 0; y < gridSize && index < count; y++ {
			for z := 0; z < gridSize && index < count; z++ {
				// Calculate position in world space
				position := math32.NewVector3(
					float32(x)*spacing-10.0, // Range [-10, 10]
					float32(y)*spacing/4,    // Range [0, 5]
					float32(z)*spacing-10.0, // Range [-10, 10]
				)

				// Create visualization
				sphereGeom := geometry.NewSphere(0.1, 8, 8)
				sphereMat := material.NewStandard(math32.NewColor("Blue"))
				sphereMesh := graphic.NewMesh(sphereGeom, sphereMat)
				sphereMesh.SetPosition(position.X, position.Y, position.Z)
				scene.Add(sphereMesh)

				particles[index] = Particle{
					X:    position.X,
					Y:    position.Y,
					Z:    position.Z,
					VX:   0,
					VY:   0,
					VZ:   0,
					Mesh: sphereMesh,
				}
				index++
			}
		}
	}
	return particles
}

func updateParticles(deltaTime float32) {
	for i := range fluidParticles {
		p := &fluidParticles[i]

		// Get vector field influence
		gridX := int((p.X + 10.0) * float32(vectorField.AreaWidth) / 20.0)
		gridY := int(p.Y * float32(vectorField.AreaHeight) / 5.0)
		gridZ := int((p.Z + 10.0) * float32(vectorField.AreaDepth) / 20.0)

		// Ensure grid coordinates are in bounds
		gridX = int(clamp(float32(gridX), 0, float32(vectorField.AreaWidth-1)))
		gridY = int(clamp(float32(gridY), 0, float32(vectorField.AreaHeight-1)))
		gridZ = int(clamp(float32(gridZ), 0, float32(vectorField.AreaDepth-1)))

		// Apply vector field velocity
		v := vectorField.Field[gridX][gridY][gridZ]
		p.VX += v.VX * deltaTime
		p.VY += v.VY * deltaTime
		p.VZ += v.VZ * deltaTime

		// Add slight turbulence
		p.VX += (rand.Float32() - 0.5) * 0.1
		p.VY += (rand.Float32() - 0.5) * 0.1
		p.VZ += (rand.Float32() - 0.5) * 0.1

		// Apply drag
		p.VX *= 0.99
		p.VY *= 0.99
		p.VZ *= 0.99

		// Update position
		p.X += p.VX * deltaTime
		p.Y += p.VY * deltaTime
		p.Z += p.VZ * deltaTime

		// Constrain to bounds
		p.X = clamp(p.X, -10, 10)
		p.Y = clamp(p.Y, 0.1, 4.9)
		p.Z = clamp(p.Z, -10, 10)

		// Update visual representation
		if p.Mesh != nil {
			p.Mesh.SetPosition(p.X, p.Y, p.Z)
		}
	}
}

func updateVectorField() {
	for x := 0; x < vectorField.AreaWidth; x++ {
		for y := 0; y < vectorField.AreaHeight; y++ {
			for z := 0; z < vectorField.AreaDepth; z++ {
				v := &vectorField.Field[x][y][z]
				v.VX_ = (v.VX + rand.Float32()*0.1) * 0.9
				v.VY_ = (v.VY + rand.Float32()*0.1) * 0.9
				v.VZ_ = (v.VZ + rand.Float32()*0.1) * 0.9

				// Limit velocity
				magnitude := calcMagnitude3D(v.VX_, v.VY_, v.VZ_)
				if magnitude > 1 {
					scale := 1 / magnitude
					v.VX_ *= scale
					v.VY_ *= scale
					v.VZ_ *= scale
				}

				v.VX = v.VX_
				v.VY = v.VY_
				v.VZ = v.VZ_
			}
		}
	}
}

func drawParticles() {
	for _, p := range fluidParticles {
		log.Printf("Particle at (%.2f, %.2f, %.2f) moving with velocity (%.2f, %.2f, %.2f)", p.X, p.Y, p.Z, p.VX, p.VY, p.VZ)
	}
}

func initializeFluidSimulation(scn *core.Node, sources []WindSource) {
	scene = scn
	windSources = sources
	vectorField = initVectorField(20, 5, 20, 10, 5, 10) // World dimensions with resolution
	fluidParticles = initParticles(1000, scene)         // Create 1000 particles in a grid
}

func simulateFluid(deltaTime float32, obstMesh *core.Node) {
	mesh = obstMesh // Update global mesh reference
	if mesh != nil {
		updateWindParticles(deltaTime, scene, mesh)
	} else {
		updateWindParticles(deltaTime, scene, nil)
	}
	updateParticles(deltaTime)
	updateVectorField()
	drawParticles()
}
