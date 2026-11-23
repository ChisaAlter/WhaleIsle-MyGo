import fs from 'node:fs'; import path from 'node:path'; import ts from 'typescript';
const SRC = path.resolve('../ui-b2336c45f2b7355c42678ce786b6f14d62b819be/apps/ui/registry/default');
const out = {};
const files = [...fs.readdirSync(path.join(SRC,'ui')).map(f=>['ui',f]), ...fs.readdirSync(path.join(SRC,'chat')).filter(f=>f.endsWith('.tsx')).map(f=>['chat',f])];
const isStr = n => ts.isStringLiteral(n) || ts.isNoSubstitutionTemplateLiteral(n);
for (const [dir,f] of files) {
  const file = path.join(SRC,dir,f); const text = fs.readFileSync(file,'utf8');
  const sf = ts.createSourceFile(f, text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  const T = n => n.getText(sf).replace(/\s+/g,' ');
  const line = n => sf.getLineAndCharacterOfPosition(n.getStart(sf)).line+1;
  const moduleVars = new Map(); const cvas = {}; const constants = {}; const components = [];
  const consumed = new Set();
  const markStr = n => consumed.add(n.getStart(sf));
  const keyOf = p => p.name ? (p.name.text ?? T(p.name)) : '';
  function parseCva(name, call){
    const [base, opts] = call.arguments; const r = {name, line: line(call), base: '', variants:{}, defaultVariants:{}, compoundVariants:[]};
    const strOf = n => { if(!n) return ''; if (isStr(n)) { markStr(n); return n.text; } if (ts.isArrayLiteralExpression(n)) return n.elements.map(strOf).join(' '); return '/*dyn*/'+T(n); };
    r.base = strOf(base);
    if (opts && ts.isObjectLiteralExpression(opts)) for (const p of opts.properties) {
      const k = keyOf(p);
      if (k==='variants') for (const vp of p.initializer.properties) { const vk=keyOf(vp); r.variants[vk]={}; for (const vv of vp.initializer.properties) r.variants[vk][keyOf(vv)] = strOf(vv.initializer); }
      if (k==='defaultVariants') for (const vp of p.initializer.properties) r.defaultVariants[keyOf(vp)] = isStr(vp.initializer)? vp.initializer.text : T(vp.initializer);
      if (k==='compoundVariants') for (const el of p.initializer.elements) { const cv={when:{},className:''}; for (const q of el.properties){ const qk=keyOf(q); if(qk==='className'||qk==='class') cv.className=strOf(q.initializer); else cv.when[qk]= isStr(q.initializer)?q.initializer.text:T(q.initializer);} r.compoundVariants.push(cv); }
    }
    return r;
  }
  for (const st of sf.statements) {
    if (ts.isVariableStatement(st)) for (const d of st.declarationList.declarations) {
      if (!d.initializer || !ts.isIdentifier(d.name)) continue;
      moduleVars.set(d.name.text, d.initializer);
      const init = d.initializer;
      if (ts.isCallExpression(init) && T(init.expression)==='cva') cvas[d.name.text] = parseCva(d.name.text, init);
      else if (isStr(init) || ts.isNumericLiteral(init)) constants[d.name.text] = init.text;
    }
  }
  function collect(expr, conds, frags, localVars, seen=new Set()){
    if (!expr) return;
    if (ts.isParenthesizedExpression(expr)||ts.isAsExpression(expr)||ts.isNonNullExpression(expr)||(ts.isSatisfiesExpression && ts.isSatisfiesExpression(expr))) return collect(expr.expression, conds, frags, localVars, seen);
    if (isStr(expr)) { markStr(expr); if (expr.text.trim()) frags.push({classes: expr.text.trim().split(/\s+/), when: conds.slice(), line: line(expr)}); return; }
    if (ts.isTemplateExpression(expr)) { const parts=[expr.head.text, ...expr.templateSpans.map(s=>s.literal.text)].join(' ').trim(); markStr(expr); if (parts) frags.push({classes: parts.split(/\s+/), when: conds.slice(), line: line(expr), template: T(expr)}); for (const s of expr.templateSpans) collect(s.expression, conds, frags, localVars, seen); return; }
    if (ts.isCallExpression(expr)) { const callee = T(expr.expression);
      if (cvas[callee]) { frags.push({cva: callee, args: expr.arguments.map(T).join(', '), when: conds.slice(), line: line(expr)}); return; }
      if (['cn','clsx','cx','twMerge'].includes(callee)) { for (const a of expr.arguments) collect(a, conds, frags, localVars, seen); return; }
      frags.push({dynamic: T(expr), when: conds.slice(), line: line(expr)}); return; }
    if (ts.isBinaryExpression(expr)) { const op = expr.operatorToken.kind;
      if (op===ts.SyntaxKind.AmpersandAmpersandToken) return collect(expr.right, [...conds, T(expr.left)], frags, localVars, seen);
      if (op===ts.SyntaxKind.BarBarToken || op===ts.SyntaxKind.QuestionQuestionToken) { collect(expr.left, conds, frags, localVars, seen); return collect(expr.right, [...conds, 'fallback of '+T(expr.left)], frags, localVars, seen); }
      if (op===ts.SyntaxKind.PlusToken) { collect(expr.left, conds, frags, localVars, seen); return collect(expr.right, conds, frags, localVars, seen); } }
    if (ts.isConditionalExpression(expr)) { collect(expr.whenTrue, [...conds, T(expr.condition)], frags, localVars, seen); return collect(expr.whenFalse, [...conds, '!('+T(expr.condition)+')'], frags, localVars, seen); }
    if (ts.isIdentifier(expr)) { const n = expr.text; if (n==='className'||n==='undefined') return; if (seen.has(n)) return; seen.add(n);
      const init = localVars.get(n) ?? moduleVars.get(n); if (init) { if (cvas[n]) return; return collect(init, conds, frags, localVars, seen); }
      frags.push({dynamic: n, when: conds.slice(), line: line(expr)}); return; }
    if (ts.isArrowFunction(expr)||ts.isFunctionExpression(expr)) { const b = expr.body; const c=[...conds, 'className-fn('+expr.parameters.map(T).join(',')+')'];
      if (ts.isBlock(b)) b.forEachChild(function w(n){ if (ts.isReturnStatement(n)) collect(n.expression, c, frags, localVars, seen); else if(!ts.isFunctionLike(n)) n.forEachChild(w); }); else collect(b, c, frags, localVars, seen); return; }
    if (ts.isObjectLiteralExpression(expr)) { for (const p of expr.properties) if (ts.isPropertyAssignment(p)) { collect(p.initializer, [...conds, 'key:'+keyOf(p)], frags, localVars, seen);} return; }
    if (ts.isArrayLiteralExpression(expr)) { for (const e of expr.elements) collect(e, conds, frags, localVars, seen); return; }
    frags.push({dynamic: T(expr), when: conds.slice(), line: line(expr)});
  }
  function attrVal(a){ if (!a.initializer) return true; if (ts.isStringLiteral(a.initializer)) return a.initializer.text; const e=a.initializer.expression; if(!e) return T(a.initializer); if (isStr(e)) return e.text; return '{'+T(e)+'}'; }
  function handleComponent(name, fn, node){
    const comp = {name, line: line(node), params:{}, parts: []};
    const p0 = fn.parameters && fn.parameters[0];
    if (p0 && ts.isObjectBindingPattern(p0.name)) for (const el of p0.name.elements) if (el.initializer) comp.params[(el.propertyName||el.name).getText(sf)] = T(el.initializer);
    const localVars = new Map();
    const visit = n => { if (ts.isVariableDeclaration(n) && ts.isIdentifier(n.name) && n.initializer) localVars.set(n.name.text, n.initializer); n.forEachChild(visit); };
    if (fn.body) visit(fn.body);
    const walk = n => {
      if (ts.isJsxOpeningElement(n)||ts.isJsxSelfClosingElement(n)) {
        const part = {tag: T(n.tagName), line: line(n), attrs:{}, fragments:[]};
        for (const a of n.attributes.properties) { if (!ts.isJsxAttribute(a)) continue; const k=a.name.getText(sf);
          if (k==='className') { const e = a.initializer && (ts.isStringLiteral(a.initializer)? a.initializer : a.initializer.expression); collect(e, [], part.fragments, localVars); }
          else if (k!=='children' && !/^on[A-Z]/.test(k) && k!=='ref' && k!=='key') { const v=attrVal(a); if (typeof v!=='string' || v.length<200) part.attrs[k]=v; } }
        if (part.fragments.length || part.attrs['data-slot']) comp.parts.push(part);
      }
      if (ts.isObjectLiteralExpression(n)) { const cp = n.properties.find(p=>ts.isPropertyAssignment(p) && keyOf(p)==='className');
        if (cp) { const part={tag:'(props object)', line: line(n), attrs:{}, fragments:[]}; for (const p of n.properties) if (ts.isPropertyAssignment(p)) { const k=keyOf(p); if(k!=='className'&&k!=='children'&&!/^on[A-Z]/.test(k)) part.attrs[k]= isStr(p.initializer)?p.initializer.text:'{'+T(p.initializer).slice(0,150)+'}'; }
          collect(cp.initializer, [], part.fragments, localVars); comp.parts.push(part);} }
      if (ts.isCallExpression(n) && T(n.expression)==='useRender') { const o=n.arguments[0]; if (o&&ts.isObjectLiteralExpression(o)) { const d=o.properties.find(p=>keyOf(p)==='defaultTagName'); if(d && d.initializer && isStr(d.initializer)) comp.defaultTagName = d.initializer.text; } }
      n.forEachChild(walk);
    };
    if (fn.body) walk(fn.body);
    components.push(comp);
  }
  for (const st of sf.statements) {
    if (ts.isFunctionDeclaration(st) && st.name) handleComponent(st.name.text, st, st);
    if (ts.isVariableStatement(st)) for (const d of st.declarationList.declarations) { let i=d.initializer; while (i && ts.isCallExpression(i) && i.arguments.length && (ts.isArrowFunction(i.arguments[0])||ts.isFunctionExpression(i.arguments[0]))) i=i.arguments[0];
      if (i && (ts.isArrowFunction(i)||ts.isFunctionExpression(i)) && ts.isIdentifier(d.name)) handleComponent(d.name.text, i, d); }
  }
  const orphans = [];
  const scan = n => { if (isStr(n) && !consumed.has(n.getStart(sf)) && n.text.length>2 && !ts.isImportDeclaration(n.parent) && !ts.isExportDeclaration(n.parent) && !ts.isExternalModuleReference(n.parent)) orphans.push({text:n.text, line: line(n), ctx: T(n.parent).slice(0,160)}); n.forEachChild(scan); };
  scan(sf);
  out[(dir==='chat'?'chat/':'')+f.replace('.tsx','')] = {file: `apps/ui/registry/default/${dir}/${f}`, cvas, constants, components, orphans};
}
fs.writeFileSync('extracted.json', JSON.stringify(out,null,1));
console.log(Object.keys(out).length, 'files');
