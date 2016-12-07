package model

import (
    "strconv"
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

type Client struct {
    clientId string
    lastTouch int64
    proxyId string
}

type Game struct {
    eventStream EventStream
    stateStream StateStream
    lastTick int64
    clients map[string]*Client
    lastProxyId int64
    mutex sync.Mutex
}

func NewGame(now int64) (game * Game) {
    return &Game{
        eventStream: NewEventStream(),
        stateStream: NewStateStream(),
        lastTick : now,
        clients : make(map[string]*Client),
        lastProxyId: 0,
    }
}

func (game * Game) getClient (now int64, clientId string) * Client {
    val, ok := game.clients[clientId]
    if (ok) {
        return val
    }
    game.lastProxyId += 1
    client := &Client{
        clientId: clientId,
        lastTouch: now,
        proxyId: strconv.FormatInt(game.lastProxyId, 10),
    }
    game.clients[clientId] = client
    game.eventStream.Connect(client.proxyId)
    return client
}

func (game * Game) Process (now int64, clientId string, lastKnownT int64, events []Event, state map[string]string) ([]Event, []State, string) {
    game.mutex.Lock()
    defer game.mutex.Unlock()
    game.tick(now)
    client := game.updateClients(now, clientId)
    game.applyEvents(events, client)
    if (state != nil) {
        game.applyState(state, client)
    }
    game.eventStream.MarkSent()
    
    deltaEvents := game.eventStream.GetDeltaEvents(lastKnownT)
    deltaStates := game.stateStream.GetDeltaState(lastKnownT)
    return deltaEvents, deltaStates, client.proxyId
}

func (game * Game) IsExpired (now int64) bool {
    return (now - game.lastTick) > GAME_TIMEOUT_MS
}


func (game * Game) tick (now int64) {
    elapsedTime := now - game.lastTick
    if (elapsedTime > STREAM_TICK_THRESHOLD_MS) {
        count := elapsedTime / STREAM_TICK_THRESHOLD_MS
        game.eventStream.Tick(count)
        game.stateStream.Tick(count)
        extra := elapsedTime % STREAM_TICK_THRESHOLD_MS
        game.lastTick = now - extra
    }
}

func (game * Game) updateClients (now int64, clientId string) * Client {
    // Expire old clients
    expired := make([]string, 0)
    for id,client := range game.clients {
        elapsed := now - client.lastTouch
        if (elapsed > CLIENT_TIMEOUT_MS) {
            expired = append(expired, id)
        }
    }
    for i := range expired {
        id := expired[i]
        proxyId := game.clients[id].proxyId
        delete(game.clients, id)
        game.stateStream.Disconnect(proxyId)
        game.eventStream.Disconnect(proxyId)
    }
    
    // Touch client
    client := game.getClient(now, clientId)
    client.lastTouch = now
    return client
}

func (game * Game) applyState (data map[string]string, client * Client) {
    game.stateStream.SetState(client.proxyId, data)
}

func (game * Game) applyEvents (events []Event, client * Client) {
    for i := range events {
        event := events[i]
        event.Origin = client.proxyId
        game.eventStream.AddEvent(event)
    }
}