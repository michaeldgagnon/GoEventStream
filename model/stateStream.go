package model

import (
)

type StateStream struct {
    T int64
    States map[string]State
}

func NewStateStream() (stream StateStream) {
    result := StateStream{
        T: 1,
        States: make(map[string]State),
    }
    return result
}

func (stream * StateStream) SetState (clientId string, data map[string]string) {
    stream.States[clientId] = NewState(stream.T, clientId, data)
}

func (stream * StateStream) Tick (count int64) {
    stream.T += count
}

func (stream * StateStream) GetDeltaState (lastKnown int64) []State {
    result := make([]State, 0)
    for _, state := range stream.States {
        if (state.T <= stream.T && (state.T > lastKnown)) {
            result = append(result, state)
        }
    }
    return result
}

func (stream * StateStream) Disconnect (clientId string) {
    delete(stream.States, clientId)
}