package model

import (
    "encoding/json"
)

type Event struct {
    T int64
    Type string
    Origin string
    Body string
}

func NewEvent(eventType string, origin string, body string) (event * Event) {
    return &Event{
        T: 0,
        Type: eventType,
        Origin: origin,
        Body: body,
    }
}

func (event * Event) SetOrigin (origin string) {
    event.Origin = origin
}

func (event * Event) SetTime (t int64) {
    event.T = t
}

func (event * Event) Serialize () (string, error) {
    b, err := json.Marshal(event)
    if (err != nil) {
        return "", err
    }
    return string(b), nil
}