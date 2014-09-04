// Copyright 2014, The cf-service-broker Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that
// can be found in the LICENSE file.

package brokerapi

import (
    "encoding/json"
    "github.com/gorilla/mux"
    "log"
    "net/http"
)

var empty struct{} = struct{}{}

type handler struct {
    manager AgentManagerInterface
}

func newHandler(am AgentManagerInterface) *handler {
    return &handler{am}
}


func (h *handler) catalog(r *http.Request) responseEntity {

    log.Printf("Handler: Requesting catalog")
    if cat, err := h.manager.Catalog(); err != nil {
        return handleServiceError(err)
    } else {
        log.Printf("Handler: Catalog retrieved",cat)
        
        return responseEntity{http.StatusOK, cat}
    }
}

func (h *handler) provision(req *http.Request) responseEntity {
    vars := mux.Vars(req)
    preq := ProvisioningRequest{InstanceId: vars[instanceId]}

    log.Printf("Handler: Provisioning: %v", preq)

    if err := json.NewDecoder(req.Body).Decode(&preq); err != nil {
        handleDecodingError(err)
    }

    log.Printf("Handler: Provisioning request decoded: %v", preq)

    var url string
    var err error

    br, err := h.manager.GetServiceAgent("")
    if (err != nil) {
        return handleServiceError(err)
    }
    url, err = br.Provision(preq)
    if err != nil {
        return handleServiceError(err)
    }

    log.Printf("Handler: Provisioned: %v", url)

    return responseEntity{http.StatusCreated, struct {
        DashboardUrl string `json:"dashboard_url"`
    }{url}}
}

func (h *handler) deprovision(req *http.Request) responseEntity {
    vars := mux.Vars(req)
    preq := ProvisioningRequest{InstanceId: vars[instanceId]}

    log.Printf("Handler: Deprovisioning: %v", preq)

    br, err := h.manager.GetServiceAgent(preq.InstanceId)
    if (err != nil) {
        return handleServiceError(err)
    }

    if err := br.Deprovision(preq); err != nil {
       return handleServiceError(err)
    }
    log.Printf("Handler: Deprovisioned: %v", preq)

    return responseEntity{http.StatusOK, empty}
}

func (h *handler) bind(req *http.Request) responseEntity {
    vars := mux.Vars(req)
    breq := BindingRequest{InstanceId: vars[instanceId], BindingId: vars[bindingId]}

    log.Printf("Handler: Binding: %v", breq)

    if err := json.NewDecoder(req.Body).Decode(&breq); err != nil {
        handleDecodingError(err)
    }

    log.Printf("Handler: Binding request decoded: %v", breq)

    br, err := h.manager.GetServiceAgent(breq.InstanceId)
    if (err != nil) {
        return handleServiceError(err)
    }

    var url string
    var cred Credentials
    
    _, cred, url, err = br.Bind(breq)
    if err != nil {
        return handleServiceError(err)
    }
    
    log.Printf("Handler: Bound: %v", cred)
    return responseEntity{http.StatusCreated, struct {
        Credentials    interface{} `json:"credentials"`
        SyslogDrainUrl string      `json:"syslog_drain_url "`
    }{cred, url}}
}

func (h *handler) unbind(req *http.Request) responseEntity {
    vars := mux.Vars(req)
    breq := BindingRequest{InstanceId: vars[instanceId], BindingId: vars[bindingId]}

    log.Printf("Handler: Unbinding: %v", breq)

    br, err := h.manager.GetServiceAgent(breq.InstanceId)
    if (err != nil) {
        return handleServiceError(err)
    }

    if err := br.Unbind(breq); err != nil {
        return handleServiceError(err)
    }

    log.Printf("Handler: Unbound: %v", breq)

    return responseEntity{http.StatusOK, empty}
}

func (h *handler)  ping(req *http.Request) responseEntity {
    var sa ServiceAgent
    var err error

    log.Printf("Handler: Ping: %v", req.Body)
    if err = json.NewDecoder(req.Body).Decode(&sa); err != nil {
        handleDecodingError(err)
    }

    err = h.manager.Ping(sa)
    if (err != nil) {
        return handleServiceError(err)
    }

    //TODO may be we should return # of containers on this ServiceAgent so that we can compute perfFactor,
    //not sure still whether agent or the broker should compute perffactor? I prefer agent as it might have necessary tools
    //locally to compute cpu,mem,disk usage and availability. But at the same time broker has the ability to compute perffactor 
    //homogenously among agents
    return responseEntity{http.StatusOK, empty}
}

func (h *handler)  addimage(req *http.Request) responseEntity {
    vars := mux.Vars(req)
    catalog := vars[catalog]
    imagename := vars[imagename]
    var img ImageDefinition
    var err error

    log.Printf("Handler: Add Image: ")

    if err = json.NewDecoder(req.Body).Decode(&img); err != nil {
        handleDecodingError(err)
    }
    log.Println(imagename,img.Name)
    err = h.manager.AddImage(catalog,img)
    if (err != nil) {
        return handleServiceError(err)
    }
    log.Printf("Handler: Add Image: %v", img)

    return responseEntity{http.StatusOK, empty}
}

func (h *handler)  addcerts(req *http.Request) responseEntity {
    vars := mux.Vars(req)
    certname := vars[certname]
    var certs BrokerCerts
    var err error

    log.Printf("Handler: Add Certs: ",certname)

    if err = json.NewDecoder(req.Body).Decode(&certs); err != nil {
        handleDecodingError(err)
    }
    err = h.manager.AddCerts(certs)
    if (err != nil) {
        return handleServiceError(err)
    }
    log.Printf("Handler: Added Certs")

    return responseEntity{http.StatusOK, empty}
}

func (h *handler)  getimage(req *http.Request) responseEntity {
    vars := mux.Vars(req)
    catalog := vars[catalog]
    imagename := vars[imagename]
    log.Printf("Handler: Get Image: ",catalog,imagename)

    img, err := h.manager.GetImage(catalog,imagename)
    if (err != nil) {
        return handleServiceError(err)
    }
    log.Printf("Handler: Get Image: %v", img)

    return responseEntity{http.StatusOK, img}
}

func (h *handler)  getcerts(req *http.Request) responseEntity {
    log.Printf("Handler: Get Certs: ")

    values := req.URL.Query()
    var name string
    if val := values.Get("name");len(val) > 0 {
        name = val
    }
    certs, err := h.manager.GetCerts(name)
    if (err != nil) {
        return handleServiceError(err)
    }
    log.Printf("Handler: Get Certs: %v", certs)

    return responseEntity{http.StatusOK, certs}
}

func (h *handler)  delimage(req *http.Request) responseEntity {
    vars := mux.Vars(req)
    catalog := vars[catalog]
    imagename := vars[imagename]
    log.Printf("Handler: Delete Image: ",catalog,imagename)

    err := h.manager.DeleteImage(catalog,imagename)
    if (err != nil) {
        return handleServiceError(err)
    }
    log.Printf("Handler: delete Image: %v", imagename)

    return responseEntity{http.StatusOK, nil}
}

func (h *handler)  delcerts(req *http.Request) responseEntity {
    vars := mux.Vars(req)
    certname := vars[certname]
    if len(certname)==0 {
        return responseEntity{http.StatusBadRequest, BrokerError{"Certficate Name is not Valid"}}
    }
    err := h.manager.DeleteCerts(certname)
    if (err != nil) {
        return handleServiceError(err)
    }
    log.Printf("Handler: Delete Certs: %v", certname)

    return responseEntity{http.StatusOK, nil}
}

func handleDecodingError(err error) responseEntity {
    log.Printf("Handler: Decoding error: %v", err)
    return responseEntity{http.StatusBadRequest, BrokerError{err.Error()}}
}

func handleServiceError(err error) responseEntity {
    log.Printf("Handler: Service error: %v", err)

    switch err := err.(type) {
    case BrokerServiceError:
        switch err.Code() {
        case ErrCodeConflict:
            return responseEntity{http.StatusConflict, BrokerError{err.Error()}}
        case ErrCodeGone:
            return responseEntity{http.StatusGone, BrokerError{err.Error()}}
        }
    }
    return responseEntity{http.StatusInternalServerError, BrokerError{err.Error()}}
}
    
