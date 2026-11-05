package pluginrpc

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// HTTP fixtures emulate the actual transport envelope as well as its result.
// Explicit identity fields (including deliberately wrong ones) stay untouched.
func fixtureResponseIdentity(request *http.Request, response *http.Response, err error) (*http.Response, error) {
	if err != nil || response == nil || response.Body == nil || request.GetBody == nil {
		return response, err
	}
	raw, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return response, err
	}
	response.Body.Close()
	response.Body = io.NopCloser(bytes.NewReader(raw))
	var envelope map[string]json.RawMessage
	if json.Unmarshal(raw, &envelope) != nil || envelope == nil {
		return response, err
	}
	body, bodyErr := request.GetBody()
	if bodyErr != nil {
		return response, err
	}
	defer body.Close()
	var req clientRequest
	if json.NewDecoder(body).Decode(&req) != nil {
		return response, err
	}
	if _, ok := envelope["type"]; !ok {
		envelope["type"] = json.RawMessage(`"server-response"`)
	}
	if _, ok := envelope["rpcId"]; !ok {
		envelope["rpcId"], _ = json.Marshal(req.RPCID)
	}
	raw, _ = json.Marshal(envelope)
	response.Body = io.NopCloser(bytes.NewReader(raw))
	return response, err
}
