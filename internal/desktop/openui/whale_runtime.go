package openui

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

// These are the owned MIT WhaleIsle 0.3.10 products, not the historical
// Open Intelligent UI application. Provenance and all dependency notices ship.
//
//go:embed viewer/v2/viewer.html
var whaleViewer string

//go:embed viewer/v2/host.cjs
var whaleHostValidator []byte

// WhaleHostValidator is the shipped Node validator matching the owned viewer.
// The desktop launcher materializes it for the managed Host's render_ui tool.
func WhaleHostValidator() []byte { return append([]byte(nil), whaleHostValidator...) }

//go:embed viewer/v2/assets/*.js
var whaleBundles embed.FS

//go:embed viewer/v2/native-adapter.js
var whaleNativeAdapter string

//go:embed viewer/v2/THIRD_PARTY_NOTICES.txt viewer/v2/LICENSE.whale-isle
var whaleNotices embed.FS

func WhaleNotices() string {
	a, _ := whaleNotices.ReadFile("viewer/v2/LICENSE.whale-isle")
	b, _ := whaleNotices.ReadFile("viewer/v2/THIRD_PARTY_NOTICES.txt")
	return string(a) + "\n\n" + string(b)
}

func IsWhaleViewer(viewer string) bool { return viewer != "" && viewer == whaleViewer }
func LoadWhaleViewer() (string, error) {
	sum := sha256.Sum256([]byte(whaleViewer))
	if hex.EncodeToString(sum[:]) != "331a36e6c801f9902464f5a9f2e8a91006aade0e488831f502b17b0caa63284d" {
		return "", errors.New("Whale UI viewer provenance mismatch")
	}
	return whaleViewer, nil
}

var bundleNames = map[string]bool{"mermaid.js": true, "three.js": true, "echarts-core.js": true, "echarts-full.js": true, "icons.js": true, "standalone-runtime.js": true}

func Bundle(name string) ([]byte, error) {
	if !bundleNames[name] {
		return nil, errors.New("unsupported Whale UI rendering bundle")
	}
	return whaleBundles.ReadFile("viewer/v2/assets/" + name)
}

func ValidLanguage(language string) bool {
	return language == "whale-ui" || language == "dsh-ui" || language == "openui" || language == "openui-lang"
}

// Adapt only the transport integration. The immutable upstream product and its
// hash remain available for provenance verification. Engines load as data: to
// fit the existing zero-network native helper, rather than opening its policy.
func isolatedWhaleViewer(viewer string) string {
	viewer = strings.Replace(viewer, "script-src 'unsafe-inline' blob:", "script-src 'unsafe-inline' data: blob:", 1)
	// Record genuine activation inside the exact admitted viewer, including async
	// export generation. The parent still requires owned bytes, replay claims,
	// current session/source, and a native save dialog before file writes.
	adapter := `<script>(()=>{let until=0;for(const type of ['click','keydown'])document.addEventListener(type,e=>{if(e.isTrusted&&(type!=='keydown'||e.key==='Enter'||e.key===' '))until=performance.now()+30000},true);globalThis.__whaleMyGoGesture=()=>performance.now()<until;})()</script>`
	viewer = strings.Replace(viewer, "<head>", "<head>"+adapter, 1)
	viewer = strings.Replace(viewer, "window.parent.postMessage({channel:NN,version:2,instanceId:t,...v},a)", "window.parent.postMessage({channel:NN,version:2,instanceId:t,...v,userGesture:globalThis.__whaleMyGoGesture?.()===true},a)", 1)
	viewer = strings.Replace(viewer, `lt=URL.createObjectURL(new Blob([new Uint8Array(Ce)],{type:"text/javascript"}))`, `lt="data:text/javascript;base64,"+btoa(Array.from(Ce,x=>String.fromCharCode(x)).join(""))`, 1)
	hooks := `let __nativeSerial=0;globalThis.__whaleMyGo={React:fa,field:co,isStreaming:Hn,post:(type,fields)=>{window.parent.postMessage({channel:'whale-ui',version:2,instanceId:window.name.slice(9),type:'native-adapter',nativeType:type,...fields},'*');return true},register:components=>{for(const c of components){if(nCt.components[c.name])nCt.components[c.name]={...nCt.components[c.name],component:c.component}}}};addEventListener('message',e=>{const d=e.data;if(e.source!==window.parent||!d||d.channel!=='whale-ui'||d.version!==2||d.instanceId!==window.name.slice(9))return;globalThis.__whaleMyGo.nativeReceive?.(d)});`
	viewer = strings.Replace(viewer, `var nCt=pE(lSe),RN=`, `var nCt=pE(lSe);`+hooks+whaleNativeAdapter+`;var RN=`, 1)
	return viewer
}

// PanelOperation carries content only. Source identity/order are assigned by
// the owning native conversation, never accepted from the child.
type PanelOperation struct {
	Append bool            `json:"append"`
	Spec   json.RawMessage `json:"spec"`
}

func ValidatePanelOperation(raw json.RawMessage) (PanelOperation, error) {
	var op PanelOperation
	var spec struct {
		Items []json.RawMessage `json:"items"`
	}
	if len(raw) > MaxEventBytes || json.Unmarshal(raw, &op) != nil || json.Unmarshal(op.Spec, &spec) != nil || spec.Items == nil {
		return op, errors.New("invalid Whale UI panel operation")
	}
	return op, nil
}
