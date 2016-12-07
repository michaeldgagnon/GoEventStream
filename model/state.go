package model

import (
    "encoding/json"
)

type State struct {
    T int64
    Origin string
    Data map[string]string
}

func NewState(t int64, origin string, data map[string]string) (state State) {
    return State{
        T: t,
        Origin: origin,
        Data: data,
    }
}

func (event * State) Serialize () (string, error) {
    b, err := json.Marshal(event)
    if (err != nil) {
        return "", err
    }
    return string(b), nil
}