package model

import (
    "sync"
)


// Tuning variables
var STREAM_TICKS_PER_SECOND int64 = 20
var GAME_TIMEOUT_SECONDS int64 = 60
var CLIENT_TIMEOUT_SECONDS int64 = 10

// Derivatives
var STREAM_TICK_THRESHOLD_MS int64 = 1000 / STREAM_TICKS_PER_SECOND
var GAME_TIMEOUT_MS int64 = 1000 * GAME_TIMEOUT_SECONDS
var CLIENT_TIMEOUT_MS int64 = 1000 * CLIENT_TIMEOUT_SECONDS

type Game struct {
    stream * EventStream
    lastTick int64
    clients map[string]int64
    mutex sync.Mutex
}

func NewGame(now int64) (game * Game) {
    return &Game{
        stream: NewStream(),
        lastTick : now,
        clients : make(map[string]int64),
    }
}

func (game * Game) Process (now int64, clientId string, lastKnownT int64, events []Event) ([]Event) {
    game.mutex.Lock()
    defer game.mutex.Unlock()
    game.tick(now)
    game.updateClients(now, clientId)
    game.applyEvents(events, clientId)
    game.stream.MarkSent()
    
    deltaEvents := game.stream.GetDeltaEvents(lastKnownT)
    return deltaEvents
}

func (game * Game) IsExpired (now int64) bool {
    return (now - game.lastTick) > GAME_TIMEOUT_MS
}


func (game * Game) tick (now int64) {
    stream := game.stream
    elapsedTime := now - game.lastTick
    if (elapsedTime > STREAM_TICK_THRESHOLD_MS) {
        stream.Tick(elapsedTime / STREAM_TICK_THRESHOLD_MS)
        extra := elapsedTime % STREAM_TICK_THRESHOLD_MS
        game.lastTick = now - extra
    }
}

func (game * Game) updateClients (now int64, clientId string) {
    // Expire old clients
    expired := make([]string, 0)
    for client,time := range game.clients {
        elapsed := now - time
        if (elapsed > CLIENT_TIMEOUT_MS) {
            expired = append(expired, client)
        }
    }
    for i := range expired {
        client := expired[i]
        delete(game.clients, client)
        game.stream.Disconnect(client)
    }
    
    // Connect new client
    _, ok := game.clients[clientId]
    if (!ok) {
        game.stream.Connect(clientId)
    }
    game.clients[clientId] = now
}

func (game * Game) applyEvents (events []Event, clientId string) {
    for i := range events {
        event := events[i]
        event.Origin = clientId
        game.stream.AddEvent(event)
    }
}