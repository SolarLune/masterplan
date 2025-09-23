package main

import (
	"time"
)

type FPSManager struct {
	startTime           time.Time
	targetFPS           float32
	targetFrameDuration time.Duration

	frameCount    int
	debugFPS      int
	debugFPSStart time.Time
}

func NewFPSManager() *FPSManager {
	m := &FPSManager{}
	m.SetTargetFPS(60)
	return m
}

func (f *FPSManager) SetTargetFPS(fps int) {
	f.targetFPS = float32(fps)
	f.targetFrameDuration = time.Second / time.Duration(fps)
}

func (f *FPSManager) TargetFPS() int {
	return int(f.targetFPS)
}

func (f *FPSManager) Start() {
	f.startTime = time.Now()
	f.frameCount++
}

func (f *FPSManager) End() {

	if time.Since(f.debugFPSStart) > time.Second {
		f.debugFPS = f.frameCount
		f.frameCount = 0
		f.debugFPSStart = time.Now()
	}

	// Simple approach works, but stutters
	// time.Sleep(f.targetFrameDuration - time.Since(f.startTime))

	// Combination pizza hut taco be-uh, sleep + busy wait
	for {

		ts := time.Since(f.startTime)

		if ts < f.targetFrameDuration/8*7 {
			time.Sleep(f.targetFrameDuration / 8)
		} else if ts >= f.targetFrameDuration {
			// Busy wait
			break
		}

	}

}

func (f *FPSManager) DebugFPS() int {
	return f.debugFPS
}

func (f *FPSManager) DeltaTime() float32 {
	return 1.0 / f.targetFPS
}

// type FPSManager struct {
// 	targetFPS float32

// 	startTime uint64

// 	frameCount    int
// 	debugFPS      int
// 	debugFPSStart time.Time
// }

// func NewFPSManager() *FPSManager {
// 	m := &FPSManager{}
// 	m.SetTargetFPS(60)
// 	return m
// }

// func (f *FPSManager) SetTargetFPS(fps int) {
// 	f.targetFPS = float32(fps)
// }

// func (f *FPSManager) TargetFPS() int {
// 	return int(f.targetFPS)
// }

// func (f *FPSManager) Start() {
// 	f.startTime = sdl.Ticks()
// 	f.frameCount++
// }

// func (f *FPSManager) End() {

// 	if time.Since(f.debugFPSStart) >= time.Second {
// 		f.debugFPS = f.frameCount
// 		f.frameCount = 0
// 		f.debugFPSStart = time.Now()
// 	}

// 	ft := 1000 / f.targetFPS

// 	for float32(sdl.Ticks()-f.startTime) < ft {
// 		time.Sleep(time.Microsecond)
// 	}

// 	f.startTime = sdl.Ticks()

// }

// func (f *FPSManager) DebugFPS() int {
// 	return f.debugFPS
// }

// func (f *FPSManager) DeltaTime() float32 {
// 	return 1.0 / f.targetFPS
// }
