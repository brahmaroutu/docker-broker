// Copyright 2014, The cf-service-broker Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that
// can be found in the LICENSE file.

package brokerapi

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
)

type Options struct {
    Host     string
    Port     int
    Username string
    Password string
    Debug    bool
    LogFile  string
    Trace    bool
    PidFile  string
}

type broker struct {
    opts   Options
    router *router
}

func New(o Options, am AgentManagerInterface) *broker {
    return &broker{o, newRouter(o, newHandler(am))}
}    

func (b *broker) Start() {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt)

    errCh := make(chan error, 1)
    go func() {
        addr := fmt.Sprintf("%v:%v", b.opts.Host, b.opts.Port)
        log.Printf("Broker started: Listening at [%v]", addr)
        errCh <- http.ListenAndServe(addr, b.router)
    }()

    select {
    case err := <-errCh:
        log.Printf("Broker shutdown with error: %v", err)
    case sig := <-sigCh:
        var _ = sig
        log.Print("Broker shutdown gracefully")
    }
}
