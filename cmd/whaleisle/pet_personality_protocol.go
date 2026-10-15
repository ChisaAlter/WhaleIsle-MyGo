package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type petPersonalityResult uint8

const (
	petPersonalityTransport petPersonalityResult = iota
	petPersonalityAccepted
	petPersonalityRejected
)

func validPetPersonality(value string) bool {
	switch value {
	case "natural", "genki", "tsundere", "poison":
		return true
	}
	return false
}

// postPetPersonality uses the original settings/update control-plane envelope.
// Transport/unmounted-route failures are retriable indefinitely; an answered
// RPC rejection has a separate bounded budget. No response body, credential,
// origin or error text is exposed to the renderer or logs.
func postPetPersonality(ctx context.Context, origin, cookie, personality string) petPersonalityResult {
	if !validPetPersonality(personality) {
		return petPersonalityRejected
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || (u.Path != "" && u.Path != "/") || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return petPersonalityTransport
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return petPersonalityTransport
	}
	rpcID := "pet-personality-" + hex.EncodeToString(nonce[:])
	body, err := json.Marshal(struct {
		Type    string `json:"type"`
		RPCID   string `json:"rpcId"`
		Method  string `json:"method"`
		Payload struct {
			Personality string `json:"personality"`
		} `json:"payload"`
	}{Type: "client-request", RPCID: rpcID, Method: "settings/update", Payload: struct {
		Personality string `json:"personality"`
	}{Personality: personality}})
	if err != nil {
		return petPersonalityTransport
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(origin, "/")+"/dsh-whale/settings/update", bytes.NewReader(body))
	if err != nil {
		return petPersonalityTransport
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	client := &http.Client{
		Timeout: 15 * time.Second,
		// The selected Host is the only destination authorized to receive its
		// cookie and this update, including on same-origin redirects.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return petPersonalityTransport
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return petPersonalityTransport
	}
	const maxReply = 1 << 20
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxReply+1))
	if err != nil || len(raw) > maxReply {
		return petPersonalityTransport
	}
	var reply struct {
		RPCID  string `json:"rpcId"`
		Result struct {
			OK    bool            `json:"ok"`
			Value json.RawMessage `json:"value"`
		} `json:"result"`
	}
	if json.Unmarshal(raw, &reply) != nil || (reply.RPCID != "" && reply.RPCID != rpcID) {
		return petPersonalityTransport
	}
	if !reply.Result.OK {
		return petPersonalityRejected
	}
	// Some plugin versions also return a business-result ok inside value.
	// An explicit negative verdict must never count as a successful mirror.
	var value struct {
		OK *bool `json:"ok"`
	}
	if len(reply.Result.Value) != 0 && string(reply.Result.Value) != "null" {
		if json.Unmarshal(reply.Result.Value, &value) != nil {
			return petPersonalityRejected
		}
		if value.OK != nil && !*value.OK {
			return petPersonalityRejected
		}
	}
	return petPersonalityAccepted
}
