package main

import (
	"math/rand"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/math32"
)

func updatePhysics(particle *WindParticle, object *core.Node, deltaTime float32) {
	if particle == nil || !particle.Alive {
		return
	}

	// Get wind field at particle position
	gridX := int((particle.Position.X + 10.0) * float32(vectorField.AreaWidth) / 20.0)
	gridY := int(particle.Position.Y * float32(vectorField.AreaHeight) / 5.0)
	gridZ := int((particle.Position.Z + 10.0) * float32(vectorField.AreaDepth) / 20.0)

	gridX = int(clamp(float32(gridX), 0, float32(vectorField.AreaWidth-1)))
	gridY = int(clamp(float32(gridY), 0, float32(vectorField.AreaHeight-1)))
	gridZ = int(clamp(float32(gridZ), 0, float32(vectorField.AreaDepth-1)))

	// Get wind velocity at this point
	v := vectorField.Field[gridX][gridY][gridZ]

	// Apply wind directly to particle velocity
	fieldStrength := float32(1.0) // Увеличиваем силу влияния ветра
	particle.Velocity.X = v.VX * fieldStrength
	particle.Velocity.Y = v.VY * fieldStrength
	particle.Velocity.Z = v.VZ * fieldStrength

	// Add small random movement
	randStrength := float32(0.1)
	particle.Velocity.X += (rand.Float32() - 0.5) * randStrength
	particle.Velocity.Y += (rand.Float32() - 0.5) * randStrength
	particle.Velocity.Z += (rand.Float32() - 0.5) * randStrength

	// Update position
	particle.Position.X += particle.Velocity.X * deltaTime
	particle.Position.Y += particle.Velocity.Y * deltaTime
	particle.Position.Z += particle.Velocity.Z * deltaTime

	// Bounce off boundaries
	if particle.Position.X < -10 || particle.Position.X > 10 {
		particle.Velocity.X *= -0.5
		particle.Position.X = clamp(particle.Position.X, -10, 10)
	}
	if particle.Position.Y < 0.1 || particle.Position.Y > 4.9 {
		particle.Velocity.Y *= -0.5
		particle.Position.Y = clamp(particle.Position.Y, 0.1, 4.9)
	}
	if particle.Position.Z < -10 || particle.Position.Z > 10 {
		particle.Velocity.Z *= -0.5
		particle.Position.Z = clamp(particle.Position.Z, -10, 10)
	}

	// Update mesh position
	if particle.Mesh != nil {
		particle.Mesh.SetPositionVec(particle.Position)
	}

	if object != nil {
		bounds := object.BoundingBox()
		if !(bounds.Min.X > bounds.Max.X || bounds.Min.Y > bounds.Max.Y || bounds.Min.Z > bounds.Max.Z) {
			particleRadius := float32(0.05) // Based on particle creation size

			// Check collision with object's bounding box
			if bounds.ContainsPoint(particle.Position) {
				// Find closest point on box surface to get collision normal
				// Manually compute closest point on box
				closest := math32.NewVector3(
					math32.Max(bounds.Min.X, math32.Min(particle.Position.X, bounds.Max.X)),
					math32.Max(bounds.Min.Y, math32.Min(particle.Position.Y, bounds.Max.Y)),
					math32.Max(bounds.Min.Z, math32.Min(particle.Position.Z, bounds.Max.Z)),
				)
				normal := closest.Sub(particle.Position).Normalize()

				// Reflect velocity with some energy loss
				particle.Velocity.Reflect(normal).MultiplyScalar(0.7)

				// Move particle out of collision
				particle.Position.Add(normal.MultiplyScalar(particleRadius))

				// Increase turbulence after collision
				particle.Turbulence = math32.Min(particle.Turbulence+0.2, 1.0)
			}
		}
	}

	// Update mesh position
	if particle.Mesh != nil {
		particle.Mesh.SetPositionVec(particle.Position)
	}
}
