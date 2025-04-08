package main

import (
	"github.com/g3n/engine/core"
	"github.com/g3n/engine/math32"
)

var velocity = math32.NewVector3(0, 0, 0)
var dragCoefficient float32 = 0.47

const airDensity = 1.225
const area = 1.0

var mass float32 = 1.0

const gravity = -9.8

func updatePhysics(particle *WindParticle, object *core.Node, objectVelocity *math32.Vector3, objectMass float32, deltaTime float32) {
	if particle == nil || object == nil || !particle.Alive {
		return
	}

	// Apply gravity to the particle
	gravityForce := math32.NewVector3(0, gravity*particle.Mass, 0)
	particle.Velocity.Add(gravityForce.MultiplyScalar(deltaTime))

	// Calculate drag force
	velocityMagnitude := particle.Velocity.Length()
	dragForce := particle.Velocity.Clone().Normalize().MultiplyScalar(
		-0.5 * airDensity * dragCoefficient * area * velocityMagnitude * velocityMagnitude,
	)
	particle.Velocity.Add(dragForce.MultiplyScalar(deltaTime))

	// Update particle position
	particle.Position.Add(particle.Velocity.Clone().MultiplyScalar(deltaTime))
	particle.Mesh.SetPositionVec(particle.Position)

	// Check interaction with the object
	objectPos := object.Position()
	distanceVec := objectPos.Sub(particle.Position)
	distance := distanceVec.Length()
	influenceRadius := float32(3.0)
	if distance > influenceRadius {
		return
	}

	// Strength of force decreases with distance (e.g., inverse-square)
	strength := 1.0 / (1.0 + distance*distance)

	// Force = wind's momentum scaled by proximity
	force := particle.Velocity.Clone().MultiplyScalar(particle.Mass * strength)

	// Apply to object's velocity
	acceleration := force.DivideScalar(objectMass)
	objectVelocity.Add(acceleration)

	// Handle collision with the object
	objectBounds := object.BoundingBox()
	if !objectBounds.Min.Equals(&objectBounds.Max) {
		center := math32.NewVector3(0, 0, 0)
		objectBounds.Center(center)
		size := math32.NewVector3(0, 0, 0)
		objectBounds.Size(size)
		halfExtents := size.MultiplyScalar(0.5)
		center.Add(&objectPos)

		if math32.Abs(particle.Position.X-center.X) < halfExtents.X &&
			math32.Abs(particle.Position.Y-center.Y) < halfExtents.Y &&
			math32.Abs(particle.Position.Z-center.Z) < halfExtents.Z {
			normal := center.Sub(particle.Position).Normalize()
			particle.Velocity.Reflect(normal).MultiplyScalar(0.7) // Bounce with reduced speed
		}
	}
}
