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
    gameMutex.Lock()
    defer gameMutex.Unlock()
    expired := make([]string, 0)
    for name,game := range games {
        if (game.IsExpired(now)) {
            expired = append(expired, name)
        }
    }
    for i := range expired {
        name := expired[i]
        delete(games, name)
    }
}

func handler (w http.ResponseWriter, r *http.Request) {
    // Parse out request
    urlParts := strings.Split(r.URL.Path, "/")
    gameName := urlParts[1]
    clientId := urlParts[2]
    lastTime := urlParts[3]
    lastKnownT, _ := strconv.ParseInt(lastTime, 10, 64)
    decoder := json.NewDecoder(r.Body)
    var req GameRequest   
    err := decoder.Decode(&req)
    if err != nil {
        panic(err)
    }
    defer r.Body.Close()
    
    now := time.Now().UnixNano() / 1000000
    
    // Get the stream
    gameMutex.Lock()
    game, ok := games[gameName]
    if (!ok) {
        game = model.NewGame(now)
        games[gameName] = game
    }
    gameMutex.Unlock()
    
    // Process the stream
    t, deltaEvents, deltaStates, proxyId := game.Process(now, clientId, lastKnownT, req.Events, req.State)
    rsp := &GameResponse{
        T : t,
        Events : deltaEvents,
        States : deltaStates,
        ProxyId : proxyId,
    }
    js, _ := rsp.Serialize()
    fmt.Fprint(w, js)
}

var games = make(map[string]*model.Game, 0)
var gameMutex sync.Mutex

type GameRequest struct {
    Events []model.Event
    State map[string]string `json:"State,omitempty"`
}

type GameResponse struct {
    T int64
    Events []model.Event
    States []model.State
    ProxyId string
}

func (rsp * GameResponse) Serialize () (string, error) {
    b, err := json.Marshal(rsp)
    if (err != nil) {
        return "", err
    }
    return string(b), nil
}