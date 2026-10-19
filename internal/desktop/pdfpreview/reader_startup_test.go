package pdfpreview

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Execute the shipped reader, rather than checking declaration ordering. A
// downstream fetch failure must reach the visible failure handler after hash
// restoration; an early initialization exception leaves the spinner forever.
func TestReaderBootstrapRestoresHashBeforeVisibleLoadFailure(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js needed to execute the embedded browser script")
	}
	source, err := readerFiles.ReadFile("reader.js")
	if err != nil {
		t.Fatal(err)
	}
	script := `const vm = require('node:vm');
const source = process.env.READER_SOURCE;
async function run(hash) {
  const elements = new Map();
  const element = key => { if (!elements.has(key)) elements.set(key, {value:'',textContent:'loading',hidden:true,addEventListener(){}}); return elements.get(key); };
  const context = vm.createContext({URLSearchParams, location:{hash}, document:{querySelector:element,addEventListener(){}}, window:{addEventListener(){}},
    ResizeObserver:class {observe(){}}, fetch:async()=>{throw new Error('owned downstream read failure')},clearTimeout,setTimeout});
  const module = new vm.SourceTextModule(source+';export const actual = {page:state.page,zoom:state.zoom,fit:state.fit,status:status.textContent,retry:retry.hidden};',{context});
  await module.link(specifier => new vm.SyntheticModule(['getDocument','PDFWorker','TextLayerBuilder'],function(){this.setExport('getDocument',()=>{});this.setExport('PDFWorker',{});this.setExport('TextLayerBuilder',class {});},{context}));
  await module.evaluate();
  return module.namespace.actual;
}
(async()=>{
  const saved=await run('#page=2&zoom=150,7,9&fit=0');
  if(saved.page!==2||saved.zoom!==150||saved.fit!==false||!saved.status.includes('owned downstream read failure')||saved.retry!==false)throw new Error(JSON.stringify(saved));
  const malformed=await run('#page=NaN&zoom=not-a-number');
  if(malformed.page!==1||malformed.zoom!==100||!malformed.status.includes('owned downstream read failure'))throw new Error(JSON.stringify(malformed));
  console.log('saved state restored; downstream failure visible; malformed hash defaults retained');
})().catch(e=>{console.error(e);process.exitCode=1;});`
	path := filepath.Join(t.TempDir(), "reader-bootstrap.cjs")
	if err := os.WriteFile(path, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, "--experimental-vm-modules", path)
	cmd.Env = append(os.Environ(), "READER_SOURCE="+string(source))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("actual reader bootstrap failed: %v\n%s", err, output)
	}
}
