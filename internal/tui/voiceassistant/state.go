// ============================================================================
// meinDENKWERK (mDW) - Lokale KI-Plattform
// ============================================================================
//
// Package:     voiceassistant
// Description: Voice Assistant - State Machine
// Author:      Mike Stoffels with Claude
// Created:     2025-12-07
// License:     MIT
// ============================================================================

package voiceassistant

import (
	"sync"
	"time"
)

// State represents the current state of the voice assistant
type State int

const (
	// StateIdle - Waiting for activation (wake word or shortcut)
	StateIdle State = iota

	// StateListening - Recording user speech
	StateListening

	// StateProcessing - Processing speech (STT + sending to mDW)
	StateProcessing

	// StateResponding - Showing/speaking response
	StateResponding

	// StateDialog - Continuous conversation mode
	StateDialog

	// StateError - Error state
	StateError
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateIdle:
		return "Bereit"
	case StateListening:
		return "Aufnahme..."
	case StateProcessing:
		return "Verarbeite..."
	case StateResponding:
		return "Antwort"
	case StateDialog:
		return "Dialog"
	case StateError:
		return "Fehler"
	default:
		return "Unbekannt"
	}
}

// StateIcon returns an icon for the state
func (s State) Icon() string {
	switch s {
	case StateIdle:
		return "⏸"
	case StateListening:
		return "🎤"
	case StateProcessing:
		return "⚙️"
	case StateResponding:
		return "💬"
	case StateDialog:
		return "🔄"
	case StateError:
		return "❌"
	default:
		return "?"
	}
}

// StateMachine manages state transitions
type StateMachine struct {
	mu            sync.RWMutex
	currentState  State
	previousState State
	stateTime     time.Time
	listeners     []StateChangeListener
}

// StateChangeListener is called when state changes
type StateChangeListener func(oldState, newState State)

// NewStateMachine creates a new state machine
func NewStateMachine() *StateMachine {
	return &StateMachine{
		currentState: StateIdle,
		stateTime:    time.Now(),
		listeners:    make([]StateChangeListener, 0),
	}
}

// Current returns the current state
func (sm *StateMachine) Current() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState
}

// Previous returns the previous state
func (sm *StateMachine) Previous() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.previousState
}

// StateTime returns when the current state was entered
func (sm *StateMachine) StateTime() time.Time {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.stateTime
}

// StateDuration returns how long we've been in the current state
func (sm *StateMachine) StateDuration() time.Duration {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return time.Since(sm.stateTime)
}

// Transition changes to a new state
func (sm *StateMachine) Transition(newState State) bool {
	sm.mu.Lock()
	oldState := sm.currentState

	// Validate transition
	if !sm.isValidTransition(oldState, newState) {
		sm.mu.Unlock()
		return false
	}

	sm.previousState = oldState
	sm.currentState = newState
	sm.stateTime = time.Now()
	listeners := sm.listeners
	sm.mu.Unlock()

	// Notify listeners
	for _, listener := range listeners {
		listener(oldState, newState)
	}

	return true
}

// AddListener adds a state change listener
func (sm *StateMachine) AddListener(listener StateChangeListener) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.listeners = append(sm.listeners, listener)
}

// isValidTransition checks if a state transition is valid
func (sm *StateMachine) isValidTransition(from, to State) bool {
	// Define valid transitions
	validTransitions := map[State][]State{
		StateIdle: {StateListening, StateError},
		StateListening: {StateProcessing, StateIdle, StateError},
		StateProcessing: {StateResponding, StateError, StateIdle},
		StateResponding: {StateIdle, StateDialog, StateListening, StateError},
		StateDialog: {StateListening, StateIdle, StateError},
		StateError: {StateIdle},
	}

	validTargets, ok := validTransitions[from]
	if !ok {
		return false
	}

	for _, valid := range validTargets {
		if valid == to {
			return true
		}
	}

	return false
}

// Reset resets the state machine to idle
func (sm *StateMachine) Reset() {
	sm.mu.Lock()
	oldState := sm.currentState
	sm.previousState = oldState
	sm.currentState = StateIdle
	sm.stateTime = time.Now()
	listeners := sm.listeners
	sm.mu.Unlock()

	for _, listener := range listeners {
		listener(oldState, StateIdle)
	}
}

// IsActive returns true if the assistant is actively doing something
func (sm *StateMachine) IsActive() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.currentState != StateIdle && sm.currentState != StateError
}
