package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"whaleisle/internal/harness"
	"whaleisle/internal/processutil"
)

const maxCompressedMetaStdout = 256 << 10

// Node's built-in decoder is the Host's session codec. It runs only for a
// compressed source log, reads that log, and emits display fields, not content.
func readCompressedSessionMeta(path string) (title, cwd string) {
	return readCompressedSessionMetaContext(context.Background(), path)
}

func readCompressedSessionMetaContext(parent context.Context, path string) (title, cwd string) {
	node, err := harness.ResolveComponentNode("")
	if err != nil {
		return "", ""
	}
	return readCompressedSessionMetaWithNode(parent, node, path)
}

func readCompressedSessionMetaWithNode(parent context.Context, node, path string) (title, cwd string) {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 || !info.Mode().IsRegular() {
		return "", ""
	}
	raw, err := runCompressedMetadataScript(parent, node, path)
	if err != nil {
		return "", ""
	}
	var meta struct{ Title, Cwd string }
	if json.Unmarshal(raw, &meta) != nil {
		return "", ""
	}
	return meta.Title, meta.Cwd
}

func runCompressedMetadataScript(parent context.Context, node, path string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	cmd := processutil.CommandContext(ctx, node, "-e", compressedMetadataScript, path)
	processutil.Background(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	raw, readErr := io.ReadAll(io.LimitReader(stdout, maxCompressedMetaStdout+1))
	_ = stdout.Close()
	if readErr != nil {
		cancel()
		_ = cmd.Wait()
		return nil, readErr
	}
	if len(raw) > maxCompressedMetaStdout {
		cancel()
		_ = cmd.Wait()
		return nil, fmt.Errorf("compressed metadata stdout exceeded %d bytes", maxCompressedMetaStdout)
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return raw, nil
}

const compressedMetadataScript = `
const fs=require('node:fs'), zlib=require('node:zlib');
const MAX_INPUT=32*1024*1024, MAX_FRAME_OUT=2*1024*1024, MAX_TOTAL_OUT=8*1024*1024, MAX_FRAMES=512;
const st=fs.lstatSync(process.argv[1]);
if(!st.isFile()||st.size>MAX_INPUT){process.stderr.write('input exceeds limit');process.exit(2);}
const b=fs.readFileSync(process.argv[1]), meta={title:'',cwd:''};
let totalOut=0, frames=0, overflow=false;
function fold(text){for(const line of text.split('\n')){try{const r=JSON.parse(line);if(r.type==='session'){if(typeof r.cwd==='string'&&r.cwd)meta.cwd=r.cwd.slice(0,1024);if(!meta.title&&typeof r.title==='string')meta.title=r.title.trim().slice(0,512);}if(r.type==='session/title'){const v=r.data?.title??r.title;if(typeof v==='string'&&v.trim())meta.title=v.trim().slice(0,512);}}catch{}}}
let p=0;
while(p+4<=b.length&&frames<MAX_FRAMES){const start=p;if(b.readUInt32LE(p)!==0xfd2fb528)break;p+=4;if(p>=b.length)break;const d=b[p++];if(d&0x18)break;const flag=d>>>6,single=!!(d&32),dict=d&3;p+=(single?0:1)+(dict===3?4:dict)+(flag===0?(single?1:0):(1<<flag));let done=false;while(p+3<=b.length){const h=b.readUIntLE(p,3);p+=3;const t=(h>>>1)&3;if(t===3){p=b.length+1;break;}p+=t===1?1:h>>>3;if(p>b.length)break;if(h&1){done=true;break;}}if(!done)break;if(d&4)p+=4;if(p>b.length)break;frames++;try{const dec=zlib.zstdDecompressSync(b.subarray(start,p),{maxOutputLength:MAX_FRAME_OUT});totalOut+=dec.length;if(totalOut>MAX_TOTAL_OUT){overflow=true;break;}fold(dec.toString('utf8'));}catch(e){if(e&&e.code==='ERR_BUFFER_TOO_LARGE'){overflow=true;break;}}}
if(overflow){process.stderr.write('decompressed output exceeds limit');process.exit(3);}
process.stdout.write(JSON.stringify(meta));
`
