package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "strconv"
    "sync"
    "time"
    "github.com/michaeldgagnon/GoEventStream/model"
)

// Tuning variables
var STREAM_TICKS_PER_SECOND int64 = 20
var STREAM_TIMEOUT_SECONDS int64 = 60
var CLIENT_TIMEOUT_SECONDS int64 = 10

// Derivatives
var STREAM_TICK_THRESHOLD_MS int64 = 1000 / STREAM_TICKS_PER_SECOND
var STREAM_TIMEOUT_MS int64 = 1000 * STREAM_TIMEOUT_SECONDS
var CLIENT_TIMEOUT_MS int64 = 1000 * CLIENT_TIMEOUT_SECONDS


func main () {
    // Stream timeout task
    ticker := time.NewTicker(1 * time.Minute)
    go func () {
        for {
            select {
                case <- ticker.C:
                    timeoutTask()
            }
        }
    }()
    
    // Set up http listener
    mux := http.NewServeMux()
    mux.HandleFunc("/", handler)
    http.ListenAndServe(":9922", mux)
}

func timeoutTask () {
    now := time.Now().UnixNano() / 1000000
    streamsMutex.Lock()
    defer streamsMutex.Unlock()
    expired := make([]string, 0)
    for name,activeStream := range streams {
        elapsed := now - activeStream.lastTick
        if (elapsed > STREAM_TIMEOUT_MS) {
            expired = append(expired, name)
        }
    }
    for i := range expired {
        name := expired[i]
        delete(streams, name)
    }
}

func handler (w http.ResponseWriter, r *http.Request) {
    // Parse out request
    urlParts := strings.Split(r.URL.Path, "/")
    streamName := urlParts[1]
    clientId := urlParts[2]
    lastTime := urlParts[3]
    decoder := json.NewDecoder(r.Body)
    var req StreamRequest   
    err := decoder.Decode(&req)
    if err != nil {
        panic(err)
    }
    defer r.Body.Close()
    
    now := time.Now().UnixNano() / 1000000
    
    // Get the stream
    streamsMutex.Lock()
    activeStream, ok := streams[streamName]
    if (!ok) {
        activeStream = &ActiveStream{
            stream: model.NewStream(),
            lastTick : now,
            clients : make(map[string]int64),
        }
        streams[streamName] = activeStream
    }
    streamsMutex.Unlock()
    
    // Process the stream
    activeStream.mutex.Lock()
    defer activeStream.mutex.Unlock()
    activeStream.tick(now)
    activeStream.updateClients(now, clientId)
    activeStream.applyEvents(req, clientId)
    activeStream.stream.MarkSent()
    
    // Return info
    lastTimeInt, _ := strconv.ParseInt(lastTime, 10, 64)
    deltaEvents := activeStream.stream.GetDeltaEvents(lastTimeInt)
    rsp := &StreamResponse{
        Events : deltaEvents,
    }
    js, _ := rsp.Serialize()
    fmt.Fprint(w, js)
}

type ActiveStream struct {
    stream * model.EventStream
    lastTick int64
    clients map[string]int64
    mutex sync.Mutex
}
var streams = make(map[string]*ActiveStream, 0)
var streamsMutex sync.Mutex

func (activeStream * ActiveStream) tick (now int64) {
    stream := activeStream.stream
    elapsedTime := now - activeStream.lastTick
    if (elapsedTime > STREAM_TICK_THRESHOLD_MS) {
        stream.Tick(elapsedTime / STREAM_TICK_THRESHOLD_MS)
        extra := elapsedTime % STREAM_TICK_THRESHOLD_MS
        activeStream.lastTick = now - extra
    }
}

func (activeStream * ActiveStream) updateClients (now int64, clientId string) {
    // Expire old clients
    expired := make([]string, 0)
    for client,time := range activeStream.clients {
        elapsed := now - time
        if (elapsed > CLIENT_TIMEOUT_MS) {
            expired = append(expired, client)
        }
    }
    for i := range expired {
        client := expired[i]
        delete(activeStream.clients, client)
        activeStream.stream.Disconnect(client)
    }
    
    // Connect new client
    _, ok := activeStream.clients[clientId]
    if (!ok) {
        activeStream.stream.Connect(clientId)
    }
    activeStream.clients[clientId] = now
}

func (activeStream * ActiveStream) applyEvents (req StreamRequest, clientId string) {
    for i := range req.Events {
        event := req.Events[i]
        event.Origin = clientId
        activeStream.stream.AddEvent(event)
    }
}

type StreamRequest struct {
    Events []model.Event
}

type StreamResponse struct {
    Events []model.Event
}

func (rsp * StreamResponse) Serialize () (string, error) {
    b, err := json.Marshal(rsp)
    if (err != nil) {
        return "", err
    }
    return string(b), nil
}