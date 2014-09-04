package brokerapi

import (
    "encoding/base64"
    "encoding/json"
    "errors"
    "fmt"
    "github.com/gorilla/mux"
    "log"
    "net/http"
    "net/http/httputil"
    "strconv"
    "strings"
)

const (
    apiVersion = "v2"
    instanceId = "iid"
    bindingId  = "bid"
    catalog    = "cat"
    imagename  = "img"
    certname   = "cert"
)

var (
    catalogUrlPattern      = fmt.Sprintf("/%v/catalog", apiVersion)
    provisioningUrlPattern = fmt.Sprintf("/%v/service_instances/{%v}", apiVersion, instanceId)
    bindingUrlPattern      = fmt.Sprintf("/%v/service_instances/{%v}/service_bindings/{%v}", apiVersion, instanceId, bindingId)
    pingUrlPattern         = fmt.Sprintf("/ping")
    imageUrlPattern        = fmt.Sprintf("/{%v}/image/{%v}",catalog,imagename)
    certUrlPattern         = fmt.Sprintf("/certificate/{%v}",certname)
    imageAllUrlPattern     = fmt.Sprintf("/{%v}/images",catalog)
    certAllUrlPattern      = fmt.Sprintf("/certificates")
)

type router struct {
    opts Options
    mux  *mux.Router // TODO: Replace with own simpler regexp-based mux???
}

func newRouter(o Options, h *handler) *router {
    mux := mux.NewRouter()
    mux.Handle(catalogUrlPattern, responseHandler(h.catalog)).Methods("GET")
    mux.Handle(provisioningUrlPattern, responseHandler(h.provision)).Methods("PUT")
    mux.Handle(provisioningUrlPattern, responseHandler(h.deprovision)).Methods("DELETE")
    mux.Handle(bindingUrlPattern, responseHandler(h.bind)).Methods("PUT")
    mux.Handle(bindingUrlPattern, responseHandler(h.unbind)).Methods("DELETE")
    mux.Handle(pingUrlPattern, responseHandler(h.ping)).Methods("POST")
    mux.Handle(imageUrlPattern, responseHandler(h.addimage)).Methods("PUT")
    mux.Handle(imageUrlPattern, responseHandler(h.getimage)).Methods("GET")
    mux.Handle(imageUrlPattern, responseHandler(h.delimage)).Methods("DELETE")
    mux.Handle(certUrlPattern, responseHandler(h.addcerts)).Methods("PUT")
    mux.Handle(certUrlPattern, responseHandler(h.getcerts)).Methods("GET")
    mux.Handle(certUrlPattern, responseHandler(h.delcerts)).Methods("DELETE")
    mux.Handle(imageAllUrlPattern, responseHandler(h.getimage)).Methods("GET")
    mux.Handle(certAllUrlPattern, responseHandler(h.getcerts)).Methods("GET")
    return &router{o, mux}
}

// Log & verify request and then pass it to Gorilla to be dispatched approprietly.
func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    if dump, err := httputil.DumpRequest(req, true); err != nil {
        log.Printf("Cannot log incoming request: %v", err)
    } else {
        if len(string(dump))<200 {
            log.Print(string(dump))
        }
    }

    major, minor, err := extractVersion(req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    log.Printf("Router: Version check: [%v.%v]", major, minor)
    //TODO: Verify compatibility

    username, password, err := extractCredentials(req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusUnauthorized)
        return
    }
    log.Printf("Router: Authentication: [%v/%v]", username, password)
    if (username != r.opts.Username) || (password != r.opts.Password) {
        log.Println("trying to login ",username," against ",r.opts.Username)
        http.Error(w, "Invalid Credentials for :"+username, http.StatusUnauthorized)
    }
    r.mux.ServeHTTP(w, req)
}

type responseEntity struct {
    status int
    value  interface{}
}

type responseHandler func(*http.Request) responseEntity

// Marshall the response entity as JSON and return the proper HTTP status code.
func (fn responseHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    re := fn(req)
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(re.status)
    if err := json.NewEncoder(w).Encode(re.value); err != nil {
        log.Printf("Error occured while marshalling response entity: %v", err)
    }
}

// Helpers
func extractVersion(req *http.Request) (int, int, error) {
    versions, _ := req.Header["X-Broker-Api-Version"]
    if len(versions) != 1 {
        return 0, 0, errors.New("Missing Broker API version")
    }
    tokens := strings.Split(versions[0], ".")
    if len(tokens) != 2 {
        return 0, 0, errors.New("Invalid Broker API version")
    }
    major, err1 := strconv.Atoi(tokens[0])
    minor, err2 := strconv.Atoi(tokens[1])
    if err1 != nil || err2 != nil {
        return 0, 0, errors.New("Invalid Broker API version")
    }
    return major, minor, nil
}

func extractCredentials(req *http.Request) (string, string, error) {
    auths, _ := req.Header["Authorization"]
    if len(auths) != 1 {
        return "", "", errors.New("Unauthorized access")
    }
    tokens := strings.Split(auths[0], " ")
    if len(tokens) != 2 || tokens[0] != "Basic" {
        return "", "", errors.New("Unsupported authentication method")
    }
    raw, err := base64.StdEncoding.DecodeString(tokens[1])
    if err != nil {
//        raw = []byte(tokens[1])
        return "", "", errors.New("Unable to decode 'Authorization' header")
    }
    credentials := strings.Split(string(raw), ":")
    if len(credentials) != 2 {
        return "", "", errors.New("Missing credentials")
    }
    return credentials[0], credentials[1], nil
}
