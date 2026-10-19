package phoneremote

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// This exercises installed cryptographic and file-backed pairing modules only.
// It does not construct DshdRemote, start a daemon, connect a relay or pair a phone.
func TestFunctionalOwnedPairingRuntimePrimitives(t *testing.T) {
	node, modules := os.Getenv("WHALEISLE_REMOTE_NODE"), os.Getenv("WHALEISLE_REMOTE_SERVER")
	if node == "" || modules == "" {
		t.Skip("set WHALEISLE_REMOTE_NODE and WHALEISLE_REMOTE_SERVER for installed-module verification")
	}
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", ownedPairingPrimitives, modules, root, l.Addr().String())
	cmd.Dir = root
	// No inherited model, Host, SSH, proxy, Node preload or account environment.
	cmd.Env = []string{"HOME=" + root, "USERPROFILE=" + root, "APPDATA=" + root, "LOCALAPPDATA=" + root,
		"TEMP=" + root, "TMP=" + root, "PATH=" + filepath.Dir(node), "SystemRoot=" + os.Getenv("SystemRoot")}
	b, err := cmd.Output()
	if err != nil {
		t.Fatalf("isolated pairing primitive process failed: %v", err)
	}
	var result struct {
		URL    string `json:"url"`
		Checks int    `json:"checks"`
	}
	if json.Unmarshal(b, &result) != nil || result.Checks != 11 {
		t.Fatal("pairing primitive checks incomplete")
	}
	if expires, ok := offerExpiry(result.URL); !ok || !expires.After(time.Now()) {
		t.Fatal("actual v2 offer rejected by Go adapter")
	}
	t.Log("actual installed modules + bundled Node: offer-v2/expiry, one-time token, synthetic device secret/HMAC/replay denial/rename/revoke/persistence; Go accepts actual offer shape; no daemon/relay/device")
}

const ownedPairingPrimitives = `
const assert=require('node:assert/strict'), path=require('node:path'), {pathToFileURL}=require('node:url');
const [modules,home,endpoint]=process.argv.slice(1);
const load=p=>import(pathToFileURL(path.join(modules,p)).href);
(async()=>{
 const {generateLocalPairingOffer}=await load('pairing-offer.js');
 const {RelayDeviceCredentialStore}=await load('relay-device-credential-store.js');
 const {computeRelayDeviceAuthProof}=await load('relay-device-auth.js');
 const generated=await generateLocalPairingOffer({chisacodeHome:home,relayEnabled:true,relayEndpoint:endpoint,relayPublicEndpoint:endpoint,relayUseTls:false,relayPublicUseTls:false,appBaseUrl:'http://'+endpoint,includeQr:false});
 assert.equal(generated.relayEnabled,true);assert.equal(generated.qr,null);
 const url=new URL(generated.url),offer=JSON.parse(Buffer.from(url.hash.slice(7),'base64url').toString());
 assert.equal(offer.v,2);assert.equal(offer.relay.endpoint,endpoint);assert.equal(offer.authBootstrap.version,1);assert(offer.authBootstrap.expiresAtMs>Date.now());
 const store=new RelayDeviceCredentialStore(home);
 assert.equal(store.consumePairingToken('invalid-synthetic-token'),false);
 assert.equal(store.consumePairingToken(offer.authBootstrap.pairingToken),true);
 assert.equal(store.consumePairingToken(offer.authBootstrap.pairingToken),false);
 const expired=store.issuePairingToken(-1000);assert.equal(store.consumePairingToken(expired.token),false);
 const device=store.issueDevice('Owned synthetic device','owned-device-01');
 assert.equal(store.verifyDeviceSecret(device.deviceId,device.deviceSecret),true);
 assert.equal(store.verifyDeviceSecret(device.deviceId,'wrong-synthetic-secret'),false);
 const fields={serverId:offer.serverId,daemonPublicKeyB64:offer.daemonPublicKeyB64,clientPublicKeyB64:'synthetic-client-public-key',deviceId:device.deviceId,challenge:'owned-first-challenge'};
 const proof=computeRelayDeviceAuthProof(device.deviceSecret,fields);
 assert.equal(store.verifyDeviceProof({...fields,proof}),true);
 assert.equal(store.verifyDeviceProof({...fields,proof}),false);
 assert.equal(store.renameDevice(device.deviceId,'Owned renamed'),true);
 assert.equal(store.listDevices()[0].label,'Owned renamed');assert.equal('secret' in store.listDevices()[0],false);
 const reopened=new RelayDeviceCredentialStore(home);assert.equal(reopened.listDevices()[0].label,'Owned renamed');
 assert.equal(reopened.revokeDevice(device.deviceId),true);
 assert.equal(new RelayDeviceCredentialStore(home).verifyDeviceSecret(device.deviceId,device.deviceSecret),false);
 process.stdout.write(JSON.stringify({url:generated.url,checks:11}));
})().catch(()=>{process.stderr.write('local pairing primitive assertion failed');process.exitCode=1});
`
