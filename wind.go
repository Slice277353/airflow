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
	Position  math32.Vector3
	Radius    float32
	Speed     float32
	Direction math32.Vector3
	Node      *graphic.Mesh
}

type WindParticle struct {
	Mesh     *graphic.Mesh
	Velocity math32.Vector3
	Position *math32.Vector3
	Lifespan float32
	Elapsed  float32
}

var windParticles []*WindParticle

func initializeWindSources(scene *core.Node) []WindSource {
	windSources := []WindSource{
		{Position: *math32.NewVector3(5, 2, 5), Radius: 3.0, Speed: 8.0, Direction: *math32.NewVector3(-1, 0, -1).Normalize()}, // Diagonal wind
		{Position: *math32.NewVector3(-5, 2, -5), Radius: 2.0, Speed: 6.0, Direction: *math32.NewVector3(1, 0, 1).Normalize()}, // Opposite diagonal
	}

	for i := range windSources {

		sphereGeom := geometry.NewSphere(0.2, 16, 16)
		sphereMat := material.NewStandard(math32.NewColor("Red"))
		sphereMesh := graphic.NewMesh(sphereGeom, sphereMat)
		sphereMesh.SetPositionVec(&windSources[i].Position)
		windSources[i].Node = sphereMesh // Store the mesh in the WindSource struct
		scene.Add(sphereMesh)
	} // a few changes in here as well

	return windSources
}

func addWindSource(windSource []WindSource, scene *core.Node, position math32.Vector3) []WindSource {
	newWind := WindSource{
		Position:  position,
		Radius:    2.0,
		Speed:     5.0,
		Direction: *math32.NewVector3(1, 0, 0).Normalize(),
	}

	sphereGeom := geometry.NewSphere(0.2, 16, 16)
	sphereMat := material.NewStandard(math32.NewColor("Red"))
	sphereMesh := graphic.NewMesh(sphereGeom, sphereMat)
	sphereMesh.SetPositionVec(&newWind.Position)
	newWind.Node = sphereMesh
	scene.Add(sphereMesh)

	return append(windSource, newWind)
}

func createWindParticle(position, direction math32.Vector3) *WindParticle {
	// Create a thin cylinder to represent wind direction
	particleGeom := geometry.NewCylinder(0.05, 0.5, 8, 1, true, true) // Use integer values for segments
	particleMat := material.NewStandard(math32.NewColor("Cyan"))      // Bright color for visibility
	particleMesh := graphic.NewMesh(particleGeom, particleMat)        // Use NewMesh instead of MeshFromGeometry

	// Position the particle
	particleMesh.SetPosition(position.X, position.Y, position.Z)

	// Calculate rotation angles directly
	yaw := math32.Atan2(direction.Z, direction.X)          // Rotation around Y-axis
	pitch := math32.Asin(direction.Y / direction.Length()) // Rotation around X-axis

	// Apply the rotation
	particleMesh.SetRotation(pitch, yaw, 0)

	log.Printf("Adding wind particle at position: %v, Direction: %v", position, direction)
	scene.Add(particleMesh)

	return &WindParticle{
		Mesh:     particleMesh,
		Velocity: *direction.Clone().MultiplyScalar(2.0), // Increase speed for visibility
		Lifespan: 5.0,
		Elapsed:  0,
	}
}

func updateWindParticles(deltaTime float32, scene *core.Node, mesh *core.Node) {
	var newParticles []*WindParticle
	log.Printf("Processing %d wind particles", len(windParticles))

	for _, particle := range windParticles {
		particle.Elapsed += deltaTime
		if particle.Elapsed >= particle.Lifespan {
			log.Printf("Removing particle at position: %v", particle.Mesh.Position())
			scene.Remove(particle.Mesh)
			continue
		}

		// Update physics for the particle
		if mesh != nil {
			updatePhysics(particle, mesh, velocity, mass, deltaTime)
		}

		// Keep particle in scene bounds (optional)
		if particle.Position.Length() > 20 {
			log.Printf("Particle out of bounds at: %v", particle.Position)
			scene.Remove(particle.Mesh)
			continue
		}

		newParticles = append(newParticles, particle)
	}

	windParticles = newParticles
}

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

func initParticles(count int, windSources []WindSource, scene *core.Node) []Particle {
	particles := make([]Particle, count)
	sourceCount := len(windSources)

	for i := 0; i < count; i++ {
		// Distribute particles evenly across wind sources
		wind := windSources[i%sourceCount]

		// Spawn particle near the wind source within its radius in 3D space
		offset := math32.NewVector3(
			(rand.Float32()-0.5)*2*wind.Radius, // X offset within the radius
			(rand.Float32()-0.5)*2*wind.Radius, // Y offset within the radius
			(rand.Float32()-0.5)*2*wind.Radius, // Z offset within the radius
		)

		// Ensure the offset is within the spherical radius
		if offset.Length() > wind.Radius {
			offset.Normalize().MultiplyScalar(wind.Radius)
		}

		position := wind.Position.Clone().Add(offset)

		// Create a small sphere for visualization
		sphereGeom := geometry.NewSphere(0.1, 8, 8)
		sphereMat := material.NewStandard(math32.NewColor("Blue"))
		sphereMesh := graphic.NewMesh(sphereGeom, sphereMat)

		// Correct positioning using SetPosition instead of SetPositionVec
		sphereMesh.SetPosition(position.X, position.Y, position.Z)
		scene.Add(sphereMesh)

		// Initialize particle velocity based on wind direction with some randomness
		velocity := wind.Direction.Clone().MultiplyScalar(wind.Speed).Add(
			math32.NewVector3(
				(rand.Float32()-0.5)*0.5,
				(rand.Float32()-0.5)*0.5, // Added Y velocity
				(rand.Float32()-0.5)*0.5,
			),
		)

		particles[i] = Particle{
			X:    position.X,
			Y:    position.Y,
			Z:    position.Z,
			VX:   velocity.X,
			VY:   velocity.Y,
			VZ:   velocity.Z,
			Mesh: sphereMesh,
		}
	}
	return particles
}

func updateParticles(deltaTime float32) {
	for i := range fluidParticles {
		p := &fluidParticles[i]

		// Random turbulence
		p.VX += (rand.Float32() - 0.5) * 0.1
		p.VY += (rand.Float32() - 0.5) * 0.1
		p.VZ += (rand.Float32() - 0.5) * 0.1

		// Friction
		p.VX *= 0.9
		p.VY *= 0.9
		p.VZ *= 0.9

		// Update position
		p.OX = p.X
		p.OY = p.Y
		p.OZ = p.Z
		p.X += p.VX * deltaTime
		p.Y += p.VY * deltaTime
		p.Z += p.VZ * deltaTime

		// Constrain to a reasonable area
		const maxX, maxY, maxZ = 10.0, 5.0, 10.0
		p.X = clamp(p.X, -maxX, maxX)
		p.Y = clamp(p.Y, 0.1, maxY) // Keep above ground, but with upper limit
		p.Z = clamp(p.Z, -maxZ, maxZ)

		// Update the sphere's position
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

func initializeFluidSimulation(scene *core.Node, windSources []WindSource) {
	vectorField = initVectorField(20, 20, 20, 10, 10, 10)   // Adjusted dimensions for better visualization
	fluidParticles = initParticles(250, windSources, scene) // Reduced particle count for clarity
}

func simulateFluid(deltaTime float32) {
	updateParticles(deltaTime)
	updateVectorField()
	drawParticles()
}
