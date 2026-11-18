package importer

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"strings"

	"whaleisle/internal/harness"
	"whaleisle/internal/processutil"
)

const maxCompressedMetaLine = 1 << 20

// sessionMeta is the display pair a session log carries: the title the catalog
// lists and the workspace its row is grouped under.
type sessionMeta struct {
	Title string
	Cwd   string
}

// A shell session log is zstd, and the only decoder for it is Node's own, so a
// title used to cost one process per session: a profile with a thousand
// sessions meant a thousand spawns. One process now decodes the whole list.
var readCompressedSessionMeta = readCompressedSessionMetaBatch

func readCompressedSessionMetaBatch(ctx context.Context, paths []string) map[string]sessionMeta {
	metas := make(map[string]sessionMeta, len(paths))
	if len(paths) == 0 || ctx.Err() != nil {
		return metas
	}
	node, err := harness.ResolveComponentNode("")
	if err != nil {
		return metas
	}
	cmd := exec.CommandContext(ctx, node, "-e", compressedMetadataBatchScript)
	processutil.Background(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return metas
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return metas
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return metas
	}
	// The whole list goes through a pipe, so it is written from its own
	// goroutine: a decoder that ever writes before it consumes cannot then
	// deadlock against a full stdin buffer.
	go func() {
		_, _ = io.WriteString(stdin, strings.Join(paths, "\n"))
		_ = stdin.Close()
	}()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64<<10), maxCompressedMetaLine)
	for scanner.Scan() {
		var row struct{ Path, Title, Cwd string }
		if json.Unmarshal(scanner.Bytes(), &row) != nil || row.Path == "" {
			continue
		}
		metas[row.Path] = sessionMeta{Title: row.Title, Cwd: row.Cwd}
	}
	_ = cmd.Wait()
	return metas
}

// The decoder reads one path per line and answers one JSON object per line.
// Each log is read independently and its own limits bound the work, so a list
// of any length still ends.
const compressedMetadataBatchScript = `
const fs=require('node:fs'), zlib=require('node:zlib');
const MAX_INPUT=32*1024*1024, MAX_FRAME_OUT=2*1024*1024, MAX_TOTAL_OUT=8*1024*1024, MAX_FRAMES=512;
function metaOf(p){
const meta={title:'',cwd:''};
try{
const st=fs.lstatSync(p);
if(!st.isFile()||st.size>MAX_INPUT)return meta;
const b=fs.readFileSync(p);
let q=0,totalOut=0,frames=0;
while(q+4<=b.length&&frames<MAX_FRAMES){
const start=q;
if(b.readUInt32LE(q)!==0xfd2fb528)break;
q+=4;
if(q>=b.length)break;
const d=b[q++];
if(d&0x18)break;
const flag=d>>>6,single=!!(d&32),dict=d&3;
q+=(single?0:1)+(dict===3?4:dict)+(flag===0?(single?1:0):(1<<flag));
let done=false;
while(q+3<=b.length){
const h=b.readUIntLE(q,3);
q+=3;
const t=(h>>>1)&3;
if(t===3){q=b.length+1;break;}
q+=t===1?1:h>>>3;
if(q>b.length)break;
if(h&1){done=true;break;}
}
if(!done)break;
if(d&4)q+=4;
if(q>b.length)break;
frames++;
let dec;
try{dec=zlib.zstdDecompressSync(b.subarray(start,q),{maxOutputLength:MAX_FRAME_OUT});}catch(e){if(e&&e.code==='ERR_BUFFER_TOO_LARGE')break;continue;}
totalOut+=dec.length;
if(totalOut>MAX_TOTAL_OUT)break;
for(const line of dec.toString('utf8').split('\n')){
let r;try{r=JSON.parse(line);}catch{continue;}
if(r.type==='session'){if(typeof r.cwd==='string'&&r.cwd)meta.cwd=r.cwd.slice(0,1024);if(!meta.title&&typeof r.title==='string')meta.title=r.title.trim().slice(0,512);}
if(r.type==='session/title'){const v=r.data?.title??r.title;if(typeof v==='string'&&v.trim())meta.title=v.trim().slice(0,512);}
}
}
}catch(e){}
return meta;
}
const paths=fs.readFileSync(0,'utf8').split('\n').filter(Boolean);
for(const p of paths){const m=metaOf(p);process.stdout.write(JSON.stringify({path:p,title:m.title,cwd:m.cwd})+'\n');}
`
