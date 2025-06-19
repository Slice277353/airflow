package main

import (
	"log"
	"math/rand"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/geometry"
	"github.com/g3n/engine/gls"
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

var (
	windSources   []WindSource
	windScene     *core.Node
	obstacleMesh  *core.Node
	windParticles []*WindParticle
	scene         *core.Node
	mesh          *core.Node
	windEnabled   bool = true
)

type WindParticle struct {
	Position    *math32.Vector3
	Velocity    *math32.Vector3
	Mass        float32
	Turbulence  float32
	Temperature float32
	Alive       bool
	Mesh        *graphic.Mesh
}

func createWindParticle(scene *core.Node, position math32.Vector3, direction math32.Vector3, speed float32, temperature float32) *WindParticle {
	sphereGeom := geometry.NewSphere(0.05, 8, 8)
	sphereMat := material.NewStandard(math32.NewColor("White"))
	sphereMesh := graphic.NewMesh(sphereGeom, sphereMat)
	sphereMesh.SetPositionVec(&position)

	particle := &WindParticle{
		Position:    position.Clone(),
		Velocity:    direction.Clone().MultiplyScalar(speed),
		Mass:        1.0,
		Turbulence:  0.1,
		Temperature: temperature,
		Alive:       true,
		Mesh:        sphereMesh,
	}

	scene.Add(sphereMesh)
	return particle
}

func updateWindParticles(deltaTime float32, scene *core.Node, object *core.Node) {
	// Update all particles
	for _, p := range windParticles {
		if p == nil || !p.Alive {
			continue
		}

		// Get vector field influence at particle position
		gridX := int((p.Position.X + 10.0) * float32(vectorField.AreaWidth) / 20.0)
		gridY := int(p.Position.Y * float32(vectorField.AreaHeight) / 5.0)
		gridZ := int((p.Position.Z + 10.0) * float32(vectorField.AreaDepth) / 20.0)

		gridX = int(clamp(float32(gridX), 0, float32(vectorField.AreaWidth-1)))
		gridY = int(clamp(float32(gridY), 0, float32(vectorField.AreaHeight-1)))
		gridZ = int(clamp(float32(gridZ), 0, float32(vectorField.AreaDepth-1)))

		// Get field velocity with stronger influence
		v := vectorField.Field[gridX][gridY][gridZ]
		fieldStrength := float32(2.0) // Tune this for wind effect

		// Apply vector field velocity additively
		p.Velocity.X += v.VX * fieldStrength * deltaTime
		p.Velocity.Y += v.VY * fieldStrength * deltaTime
		p.Velocity.Z += v.VZ * fieldStrength * deltaTime

		// Add some turbulence for more natural movement
		turbulence := float32(0.2) // Tune this for randomness
		p.Velocity.X += (rand.Float32() - 0.5) * turbulence
		p.Velocity.Y += (rand.Float32() - 0.5) * turbulence
		p.Velocity.Z += (rand.Float32() - 0.5) * turbulence

		// Apply drag to prevent excessive speeds
		drag := float32(0.98) // Reduced drag to allow more movement
		p.Velocity.MultiplyScalar(drag)

		// Update position
		p.Position.X += p.Velocity.X * deltaTime
		p.Position.Y += p.Velocity.Y * deltaTime
		p.Position.Z += p.Velocity.Z * deltaTime

		// Mesh collision (triangle-based)
		if object != nil {
			collided, closest := checkParticleMeshCollisionRecursive(p, object, 0.05)
			if collided && closest != nil {
				normal := p.Position.Clone().Sub(closest).Normalize()
				// Reflect velocity
				p.Velocity.Reflect(normal).MultiplyScalar(0.7)
				// Move particle out of collision
				p.Position.Add(normal.MultiplyScalar(0.05))
				p.Turbulence = math32.Min(p.Turbulence+0.2, 1.0)
			}
		}

		// Boundary constraints with bounce and keep particles in bounds
		if p.Position.X < -10 || p.Position.X > 10 {
			p.Velocity.X *= -0.8 // More elastic bounce
			p.Position.X = clamp(p.Position.X, -10, 10)
		}
		if p.Position.Y < 0.1 || p.Position.Y > 4.9 {
			p.Velocity.Y *= -0.8
			p.Position.Y = clamp(p.Position.Y, 0.1, 4.9)
		}
		if p.Position.Z < -10 || p.Position.Z > 10 {
			p.Velocity.Z *= -0.8
			p.Position.Z = clamp(p.Position.Z, -10, 10)
		}

		// Update mesh position
		if p.Mesh != nil {
			p.Mesh.SetPositionVec(p.Position)
		}
	}
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

func updateVectorFieldFromSource(source *WindSource) {
	// Convert world position to grid coordinates
	gridX := int((source.Position.X + 10.0) * float32(vectorField.AreaWidth) / 20.0)
	gridY := int(source.Position.Y * float32(vectorField.AreaHeight) / 5.0)
	gridZ := int((source.Position.Z + 10.0) * float32(vectorField.AreaDepth) / 20.0)

	// Update surrounding grid points with stronger influence
	radius := int(source.Radius * float32(vectorField.AreaWidth) / 20.0)
	for x := gridX - radius; x <= gridX+radius; x++ {
		for y := gridY - radius; y <= gridY+radius; y++ {
			for z := gridZ - radius; z <= gridZ+radius; z++ {
				if x < 0 || x >= vectorField.AreaWidth ||
					y < 0 || y >= vectorField.AreaHeight ||
					z < 0 || z >= vectorField.AreaDepth {
					continue
				}

				worldPos := math32.NewVector3(
					float32(x)*20.0/float32(vectorField.AreaWidth)-10.0,
					float32(y)*5.0/float32(vectorField.AreaHeight),
					float32(z)*20.0/float32(vectorField.AreaDepth)-10.0,
				)
				toPoint := worldPos.Sub(&source.Position)
				distance := toPoint.Length()

				if distance <= source.Radius {
					// Changed influence calculation for more dynamic effect
					influence := 1.0 - math32.Pow(distance/source.Radius, 2)
					windVector := source.Direction.Clone().MultiplyScalar(influence * source.Speed * 0.01)

					// Stronger temperature influence
					tempInfluence := (source.Temperature - 20.0) * 0.2
					windVector.Y += tempInfluence

					// Set vector field values
					cell := &vectorField.Field[x][y][z]
					cell.VX = windVector.X
					cell.VY = windVector.Y
					cell.VZ = windVector.Z

					// Scale turbulence with speed
					turbulence := source.Speed * 0.002
					cell.VX += (rand.Float32() - 0.5) * turbulence
					cell.VY += (rand.Float32() - 0.5) * turbulence
					cell.VZ += (rand.Float32() - 0.5) * turbulence
				}
			}
		}
	}
}

func initializeWindSources(scn *core.Node) []WindSource {
	windScene = scn
	windSources = []WindSource{
		{Position: *math32.NewVector3(5, 2, 5), Radius: 3.0, Speed: 5.0, Direction: *math32.NewVector3(-1, 0, -1).Normalize(), Spread: 0.2, Temperature: 25.0},
		{Position: *math32.NewVector3(-5, 2, -5), Radius: 2.0, Speed: 3.0, Direction: *math32.NewVector3(1, 0, 1).Normalize(), Spread: 0.3, Temperature: 20.0},
	}

	// Initialize unified vector field with higher resolution
	vectorField = initVectorField(20, 5, 20, 40, 10, 40)

	for i := range windSources {
		sphereGeom := geometry.NewSphere(0.2, 16, 16)
		sphereMat := material.NewStandard(math32.NewColor("Red"))
		sphereMesh := graphic.NewMesh(sphereGeom, sphereMat)
		sphereMesh.SetPositionVec(&windSources[i].Position)
		windSources[i].Node = sphereMesh
		scene.Add(sphereMesh)

		updateVectorFieldFromSource(&windSources[i])
	}

	return windSources
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

func updateParticles(deltaTime float32) {
	// Only update if wind is enabled
	if !windEnabled {
		return
	}

	for i := range fluidParticles {
		p := &fluidParticles[i]

		// Get vector field influence at particle position
		gridX := int((p.X + 10.0) * float32(vectorField.AreaWidth) / 20.0)
		gridY := int(p.Y * float32(vectorField.AreaHeight) / 5.0)
		gridZ := int((p.Z + 10.0) * float32(vectorField.AreaDepth) / 20.0)

		gridX = int(clamp(float32(gridX), 0, float32(vectorField.AreaWidth-1)))
		gridY = int(clamp(float32(gridY), 0, float32((vectorField.AreaHeight - 1))))
		gridZ = int(clamp(float32(gridZ), 0, float32(vectorField.AreaDepth-1)))

		// Get field velocity
		v := vectorField.Field[gridX][gridY][gridZ]

		// Apply vector field velocity directly with reduced magnitude
		fieldStrength := float32(0.1) // Adjust this to control overall influence
		p.VX = v.VX * fieldStrength
		p.VY = v.VY * fieldStrength
		p.VZ = v.VZ * fieldStrength

		// Apply slight drag to prevent excessive speeds
		drag := float32(0.99)
		p.VX *= drag
		p.VY *= drag
		p.VZ *= drag

		// Update position
		p.X += p.VX * deltaTime
		p.Y += p.VY * deltaTime
		p.Z += p.VZ * deltaTime

		// Constrain to bounds with bounce
		if p.X < -10 || p.X > 10 {
			p.VX *= -0.5
			p.X = clamp(p.X, -10, 10)
		}
		if p.Y < 0.1 || p.Y > 4.9 {
			p.VY *= -0.5
			p.Y = clamp(p.Y, 0.1, 4.9)
		}
		if p.Z < -10 || p.Z > 10 {
			p.VZ *= -0.5
			p.Z = clamp(p.Z, -10, 10)
		}

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

func initializeFluidSimulation(scn *core.Node, sources []WindSource) {
	scene = scn
	windSources = sources
	vectorField = initVectorField(20, 5, 20, 10, 5, 10)

	// Initialize fluid particles
	// fluidParticles = initParticles(10000, scene)

	// Create initial wind particles
	windParticles = nil // Clear any existing particles

	// Create particles for each source
	particlesPerSource := 500 // Increased for better data collection
	for _, source := range sources {
		for i := 0; i < particlesPerSource; i++ {
			particle := createWindParticle(scene, source.Position, source.Direction, source.Speed, source.Temperature)
			if particle != nil {
				windParticles = append(windParticles, particle)
			}
		}
	}

	log.Printf("Initialized simulation with %d wind sources, %d wind particles, and %d fluid particles",
		len(sources), len(windParticles), len(fluidParticles))
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
}

func clearFluidParticles(scene *core.Node) {
	for _, p := range fluidParticles {
		if p.Mesh != nil {
			scene.Remove(p.Mesh)
		}
	}
	fluidParticles = nil
}

func addWindSource(sources []WindSource, scene *core.Node, position math32.Vector3) []WindSource {
	// Create new wind source with default values
	newSource := WindSource{
		Position:    position,
		Radius:      2.0,
		Speed:       3.0,
		Direction:   *math32.NewVector3(1, 0, 0).Normalize(),
		Spread:      0.2,
		Temperature: 20.0,
	}

	// Create visual representation
	sphereGeom := geometry.NewSphere(0.2, 16, 16)
	sphereMat := material.NewStandard(math32.NewColor("Red"))
	sphereMesh := graphic.NewMesh(sphereGeom, sphereMat)
	sphereMesh.SetPositionVec(&position)
	newSource.Node = sphereMesh
	scene.Add(sphereMesh)

	// Add to sources and update vector field
	sources = append(sources, newSource)
	updateVectorFieldFromSource(&newSource)

	return sources
}

// Recursive mesh collision function for groups and meshes, with world transform
func checkParticleMeshCollisionRecursive(particle *WindParticle, node core.INode, particleRadius float32) (bool, *math32.Vector3) {
	// If this node is a mesh, check collision
	if mesh, ok := node.(*graphic.Mesh); ok {
		geom := mesh.GetGeometry()
		if geom != nil {
			posAttr := geom.VBO(gls.VertexPosition)
			if posAttr != nil {
				positions := posAttr.Buffer().ToFloat32()
				indices := geom.Indices()
				worldMatrix := mesh.ModelMatrix()
				if len(indices) == 0 {
					for i := 0; i+2 < len(positions)/3; i += 3 {
						a := math32.NewVector3(positions[3*i+0], positions[3*i+1], positions[3*i+2]).ApplyMatrix4(worldMatrix)
						b := math32.NewVector3(positions[3*(i+1)+0], positions[3*(i+1)+1], positions[3*(i+1)+2]).ApplyMatrix4(worldMatrix)
						c := math32.NewVector3(positions[3*(i+2)+0], positions[3*(i+2)+1], positions[3*(i+2)+2]).ApplyMatrix4(worldMatrix)
						dist, closest := pointToTriangleDistance(particle.Position, a, b, c)
						if dist < particleRadius {
							return true, closest
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
						dist, closest := pointToTriangleDistance(particle.Position, a, b, c)
						if dist < particleRadius {
							return true, closest
						}
					}
				}
			}
		}
	}
	// If this node is a group, check all children
	if group, ok := node.(*core.Node); ok {
		for _, child := range group.Children() {
			if collided, closest := checkParticleMeshCollisionRecursive(particle, child, particleRadius); collided {
				return true, closest
			}
		}
	}
	return false, nil
}

func pointToTriangleDistance(p, a, b, c *math32.Vector3) (float32, *math32.Vector3) {
	ab := b.Clone().Sub(a)
	ac := c.Clone().Sub(a)
	ap := p.Clone().Sub(a)

	dotABAB := ab.Dot(ab)
	dotABAC := ab.Dot(ac)
	dotACAC := ac.Dot(ac)
	dotAPAB := ap.Dot(ab)
	dotAPAC := ap.Dot(ac)

	denom := dotABAB*dotACAC - dotABAC*dotABAC
	if denom == 0 {
		return ap.Length(), a // Degenerate triangle
	}

	u := (dotACAC*dotAPAB - dotABAC*dotAPAC) / denom
	v := (dotABAB*dotAPAC - dotABAC*dotAPAB) / denom

	if u >= 0 && v >= 0 && (u+v) <= 1 {
		closest := a.Clone().Add(ab.Clone().MultiplyScalar(u)).Add(ac.Clone().MultiplyScalar(v))
		return p.DistanceTo(closest), closest
	}
	// Clamp to closest edge or vertex
	minDist := float32(1e9)
	closest := a
	for _, edge := range [][2]*math32.Vector3{{a, b}, {b, c}, {c, a}} {
		d, q := pointToSegmentDistance(p, edge[0], edge[1])
		if d < minDist {
			minDist = d
			closest = q
		}
	}
	return minDist, closest
}

func pointToSegmentDistance(p, a, b *math32.Vector3) (float32, *math32.Vector3) {
	ab := b.Clone().Sub(a)
	ap := p.Clone().Sub(a)
	len2 := ab.LengthSq()
	if len2 == 0 {
		return ap.Length(), a
	}
	t := ap.Dot(ab) / len2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	closest := a.Clone().Add(ab.MultiplyScalar(t))
	return p.DistanceTo(closest), closest
}
