package main

import (
	"math/rand"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/math32"
)

const (
	airDensity       = 1.225
	dragCoefficient  = 0.47
	area             = 1.0
	gravity          = -9.8
	buoyancyFactor   = 0.1
	turbulenceFactor = 0.5
	thermalDiffusion = 0.02
)

func updatePhysics(particle *WindParticle, object *core.Node, deltaTime float32) {
	if particle == nil || !particle.Alive {
		return
	}

	// Apply gravity
	gravityForce := math32.NewVector3(0, gravity*particle.Mass, 0)

	// Apply buoyancy based on temperature difference
	temperatureDiff := particle.Temperature - 20.0
	buoyancyForce := math32.NewVector3(0, temperatureDiff*buoyancyFactor, 0)

	// Apply turbulence
	turbulence := math32.NewVector3(
		(rand.Float32()-0.5)*turbulenceFactor*particle.Turbulence,
		(rand.Float32()-0.5)*turbulenceFactor*particle.Turbulence,
		(rand.Float32()-0.5)*turbulenceFactor*particle.Turbulence,
	)

	// Calculate total force
	totalForce := gravityForce.Add(buoyancyForce).Add(turbulence)

	// Apply drag force
	velocity := particle.Velocity.Length()
	dragForce := particle.Velocity.Clone().Normalize().MultiplyScalar(
		-0.5 * airDensity * dragCoefficient * area * velocity * velocity,
	)
	totalForce.Add(dragForce)

	// Update velocity using forces
	acceleration := totalForce.MultiplyScalar(1.0 / particle.Mass)
	particle.Velocity.Add(acceleration.MultiplyScalar(deltaTime))

	// Temperature diffusion
	particle.Temperature += (20.0 - particle.Temperature) * thermalDiffusion * deltaTime

	// Update position
	movement := particle.Velocity.Clone().MultiplyScalar(deltaTime)
	particle.Position.Add(movement)
	particle.Mesh.SetPositionVec(particle.Position)

	// Update particle orientation to match velocity direction
	if particle.Velocity.Length() > 0.01 {
		direction := particle.Velocity.Clone().Normalize()
		yaw := math32.Atan2(direction.Z, direction.X)
		pitch := math32.Asin(direction.Y / direction.Length())
		particle.Mesh.SetRotation(pitch, yaw, 0)
	}

	// Handle collision with objects
	if object != nil {
		objectBounds := object.BoundingBox()
		center := math32.NewVector3(0, 0, 0)
		size := math32.NewVector3(0, 0, 0)

		// Check if box has valid dimensions by checking if min != max
		if !objectBounds.Min.Equals(&objectBounds.Max) {
			objectBounds.Center(center)
			objectBounds.Size(size)
			halfExtents := size.MultiplyScalar(0.5)
			objectPos := object.Position()
			center.Add(&objectPos) // Convert Position() to pointer by taking address

			// Check for collision
			if math32.Abs(particle.Position.X-center.X) < halfExtents.X &&
				math32.Abs(particle.Position.Y-center.Y) < halfExtents.Y &&
				math32.Abs(particle.Position.Z-center.Z) < halfExtents.Z {

				// Calculate collision normal
				normal := particle.Position.Clone().Sub(center).Normalize()

				// Reflect velocity with energy loss
				particle.Velocity.Reflect(normal).MultiplyScalar(0.7)

				// Move particle out of collision
				escapeVector := normal.MultiplyScalar(0.1)
				particle.Position.Add(escapeVector)

				// Add some turbulence after collision
				particle.Turbulence += 0.2
			}
		}
	}
}
