# GoEventStream
Networked simulation synchronization via simple shared event stream model
Running on 52.20.33.80:9922
To test, post "{}" to http://52.20.33.80:9922/foo/me/0
Where 'foo' is the game name, 'me' is your unique client id, and '0' is your last known time


## Overview
- EventStream is based around the concept of distributed deterministic simulations. The server contains no state other than a stream of messages that yield the same simulation result when processed. As such, the primary risk of misbehaving clients (bugs or malicious) is getting out of sync such that the misbehaving client's view will show them something that differs from what all other clients see.
- The server owns the current simulation time 't'
- The server owns the list of all events and the time at which they were posted.
- The server is the only thing that can ultimately put events into the stream. When an event is put into the stream, it is assigned the current time 't'
- Events have a time, type, origin, and body. Time indicates the tick in which the event occurs. Type describes what the event is. Origin defines who created it ('_' for server generated). Body is an arbitrary string content.
- The server always puts in initial event '_a' at time 0 with a random body. This is used to all clients to synchronize on the same random seed
- Clients connect and disconnect from a stream at any time. Whenever a connection event happens, the server puts a connect '_c' or disconnect '_d' event into the stream at the current time 't' with a body set to the connecting client 'public id'
- Client connect events are considered to happen whenever the server encounters a private client id that it does not already know about
- Client disconnect events are considered to happen whenever a known private client id is not heard from for 10 seconds
- Clients post to URLs of the form [streamId]/[clientPrivateId]/[lastKnownTick]
- The server response to all messages with the client's assigned 'public id', the current server tick, and the set of events that occur between the lastKnownTick and current server tick. The body must contain any events the client wishes to publish to the stream
- Clients must never process their simulation past the last known 't' received from the server. If they ever reach that time, they must discontinue processing
- Clients should strive for a tick rate of 20 ticks per second to minimzie the frequency of having to pause or catch up
- Clients should strive for an average Sync rate with the server of about 5 times per second. At this rate, a client should average receiving about 4 ticks at a time per sync
- Clients should tick more than 20 times per second when they are 'behind'. A healthy tolerance is to play catch up whenever the current client sim is more than 4 ticks (~1 sync) behind the current max known time 't'
- A tick approximates 50ms of wall time
- A sync approximates receiving 200ms of wall time
- The tolerance catchup approximates staying no more than 200ms behind real time
- This strategy results in a client simulation which is running consistently 200ms behind real time without any perceivable latency (since these guidelines dynamically smooth it when present)

## Stream Names
- A single server hosts potentially many streams. A stream is identified only by it's name
- Different game sessions should all have different names.
- 'Global' game lobbies may be achieved by using constant well known names
- 'Dynamic' game sessions on the fly may be achieved by using randomized lobby names (such as a guid)
- Game client version isolation may be achieved by strategically structuring the stream name. If clients concatenate a version to all stream names, it will naturally isolate them to only ever communicating with other compatible clients

## Client names
- All clients should have a unique id they decide upon and maintain themselves. This is the 'private id'. The EventStream client internally saves/loads this and generates it whenever it is not found.
- Every stream holds a mapping of 'private id' to 'public id'. Whenever a new 'private id' is encountered, the server will generate a new 'public id' for it. This mapping is different in every stream and has no persistence of any sort
- When a client connects, they receive their public id. This enables them to identify themselves from now on in all events being processed.
- When a client reconnects to the same stream after disconnecting, as long as they gave the same private id, they will get the same public id


# GameSim
Distributed game simulation client framework on top of EventStream

- GameSim is a thin client library layer on top of EventStream to provide more clear game driving interfaces for client integration against an event stream server
- SimDriver wraps the EventStream client in a driver which then calls back on convenient interfaces to drive the simulation from the stream
- SimDriver provides convnience functions for spawning simulation-aware entities (SimEntities)
- SimDriver may only attempt to Spawn a SimEntity during the processing of a Tick or Event (to maintain determinism)
- Whenever a SimDriver is requested to spawn a SimEntity, they will only be created and added to the scene at the end of the next tick (to maintain determinism)
- SimDriver contains a single Random value which should be used for all scenarios where Random is desired
- SimDriver 'GetRandom' should be used whenever a Random value is desired
- Whenever SimDriver processes an EventStream tick, it updates the Random value and then calls 'SimTick' on all 'SimEntities' currently in the scene

- SimDriver provides a SendEvent interface which will take an event type and body. It will queue this to be pushed out in the next sync.
- When a simulation sends an event, it should not try to process that event until it comes back around in OnClientEvent

## Callback Interface
- 'OnReset' is invoked whenever a '_a' is encoutnered in the stream (always the first event). The origin is always '_'. Additionally, the SimDriver will seed Random from the guid in the _a event. In general, a simulation should destroy everything when it sees this and start the scene from scratch.
- 'OnConnect' is invoked whenever a '_c' event is encountered in the stream. The origin is always '_'. The body is always the public id of the client who connected
- 'OnDisconnect' is invoked whenever a '_d' event is encountered in the stream. The origin is always '_'. The body is always the public id of the client who disconnected.
- 'OnClientEvent' is invoked whenever any other event is encountered in the stream. The origin is always the public id of the client who sent it. The body is whatever body the client provided
