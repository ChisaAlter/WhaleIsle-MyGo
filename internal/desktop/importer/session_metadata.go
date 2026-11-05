package importer

import (
	"context"
	"encoding/json"
	"os/exec"
	"time"

	"whaleisle/internal/harness"
	"whaleisle/internal/processutil"
)

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
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", compressedMetadataScript, path)
	processutil.Background(cmd)
	raw, err := cmd.Output()
	if err != nil {
		return "", ""
	}
	var meta struct{ Title, Cwd string }
	if json.Unmarshal(raw, &meta) != nil {
		return "", ""
	}
	return meta.Title, meta.Cwd
}

const compressedMetadataScript = `
const fs=require('node:fs'), zlib=require('node:zlib');
const b=fs.readFileSync(process.argv[1]), meta={title:'',cwd:''};
function fold(text){for(const line of text.split('\n')){try{const r=JSON.parse(line);if(r.type==='session'){if(typeof r.cwd==='string'&&r.cwd)meta.cwd=r.cwd;if(!meta.title&&typeof r.title==='string')meta.title=r.title.trim();}if(r.type==='session/title'){const v=r.data?.title??r.title;if(typeof v==='string'&&v.trim())meta.title=v.trim();}}catch{}}}
let p=0;
while(p+4<=b.length){const start=p;if(b.readUInt32LE(p)!==0xfd2fb528)break;p+=4;if(p>=b.length)break;const d=b[p++];if(d&0x18)break;const flag=d>>>6,single=!!(d&32),dict=d&3;p+=(single?0:1)+(dict===3?4:dict)+(flag===0?(single?1:0):(1<<flag));let done=false;while(p+3<=b.length){const h=b.readUIntLE(p,3);p+=3;const t=(h>>>1)&3;if(t===3){p=b.length+1;break;}p+=t===1?1:h>>>3;if(p>b.length)break;if(h&1){done=true;break;}}if(!done)break;if(d&4)p+=4;if(p>b.length)break;try{fold(zlib.zstdDecompressSync(b.subarray(start,p)).toString('utf8'));}catch{}}
process.stdout.write(JSON.stringify(meta));
`
