package main

import (
	"log"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/math32"
)

var velocity = math32.NewVector3(0, 0, 0)
var dragCoefficient float32 = 0.47

const airDensity = 1.225
const area = 1.0

var mass float32 = 1.0

const gravity = -9.8

func updatePhysics(mesh *core.Node, windSources []WindSource, dt float32) {
	torusPos := mesh.Position()

	totalForce := math32.NewVector3(0, 0, 0)
	angularMomentum := math32.NewVector3(0, 0, 0)
	windPower := float32(0)
	dampingEffect := float32(0.01)

	for i := range windSources {
		wind := &windSources[i]
		distanceVec := torusPos.Clone().Sub(&wind.Position)
		distance := distanceVec.Length()

		if distance <= wind.Radius {
			windVelocity := wind.Direction.Clone().MultiplyScalar(wind.Speed)
			dragMagnitude := 0.5 * airDensity * wind.Speed * wind.Speed * dragCoefficient * area
			dragForce := windVelocity.Clone().Normalize().MultiplyScalar(dragMagnitude)
			totalForce.Add(dragForce)

			windPower += dragMagnitude * wind.Speed
			angularMomentum.Add(dragForce.Cross(&torusPos))

			windParticles = append(windParticles, createWindParticle(wind.Position, wind.Direction))
		}
	}

	gravityForce := math32.NewVector3(0, gravity*mass, 0)
	totalForce.Add(gravityForce)

	velocity.MultiplyScalar(1 - dampingEffect)
	acceleration := totalForce.DivideScalar(mass)
	velocity.Add(acceleration.MultiplyScalar(dt))

	if velocity.Length() > 10 {
		velocity.Normalize().MultiplyScalar(10)
	}

	//Temporarily disable position update to test disappearance
	displacement := velocity.Clone().MultiplyScalar(dt)
	newPos := torusPos.Add(displacement)
	if newPos.Length() > 20 {
		newPos.Normalize().MultiplyScalar(20)
	}
	if newPos.Y < 1 {
		newPos.SetY(1)
		velocity.SetY(0)
	}
	mesh.SetPositionVec(newPos)

	log.Printf("Physics update - Calculated position: %v, Velocity: %v", torusPos, velocity)

	recordSimulationData(dt, *acceleration, windPower, *angularMomentum, dampingEffect)
}
