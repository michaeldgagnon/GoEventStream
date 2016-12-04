package model

import (
    "encoding/json"
    "math/rand"
    "strconv"
    "time"
)

type EventStream struct {
    StartTime int64
    EndTime int64
    StreamSeq int
    EndWait bool
    LastSentT int64
    T int64
    Events []Event
    ProxyIds map[string]string
    LastProxyId int64
}

func NewStream() (stream * EventStream) {
    result := &EventStream{
        StartTime: 0,
        EndTime: 0,
        StreamSeq: 0,
        EndWait: false,
        LastSentT: 0,
        T: 0,
        Events: make([]Event, 0),
        ProxyIds: make(map[string]string),
        LastProxyId: 0,
    }
    result.Restart()
    return result
}

func (stream * EventStream) Restart () {
    stream.StartTime = time.Now().UnixNano() / 1000000
    stream.StreamSeq += 1
    stream.EndWait = false
    stream.LastSentT = 0
    stream.T = 0
    stream.Events = make([]Event, 0) 
    stream.ProxyIds = make(map[string]string)
    stream.AddEvent(*NewEvent("_a", "_", strconv.FormatInt(rand.Int63(), 10)))
}

func (stream * EventStream) MarkEnd () {
    stream.EndWait = true
}

func (stream * EventStream) AddEvent (event Event) {
    if (stream.EndWait) {
        return
    }
    event.SetTime(stream.LastSentT + 1)
    stream.Events = append(stream.Events, event)
}

func (stream * EventStream) GetProxyId (origin string) string {
    val, ok := stream.ProxyIds[origin]
    if (ok) {
        return val
    }
    stream.LastProxyId += 1
    newVal := strconv.FormatInt(stream.LastProxyId, 10)
    stream.ProxyIds[origin] = newVal
    return newVal
}

func (stream * EventStream) Tick (count int64) {
    if (stream.EndWait) {
        return
    }
    stream.T += count
}

func (stream * EventStream) GetDeltaEvents (lastKnown int64) []Event {
    result := make([]Event, 0)
    for _, event := range stream.Events {
        if (event.T <= stream.T && (event.T > lastKnown)) {
            result = append(result, event)
        }
    }
    return result
}

func (stream * EventStream) MarkSent () {
    stream.LastSentT = stream.T
}

func (stream * EventStream) Disconnect (clientId string) {
    proxyId := stream.GetProxyId(clientId)
    stream.AddEvent(*NewEvent("_d", "_", proxyId))
}

func (stream * EventStream) Connect (clientId string) {
    proxyId := stream.GetProxyId(clientId)
    stream.AddEvent(*NewEvent("_c", "_", proxyId))
}

func (stream * EventStream) Serialize () (string, error) {
    b, err := json.Marshal(stream)
    if (err != nil) {
        return "", err
    }
    return string(b), nil
}