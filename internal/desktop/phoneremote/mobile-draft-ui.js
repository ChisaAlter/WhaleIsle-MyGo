// Appended to the installed mobile app module by MyGo. All requests retain the
// existing authenticated gitRpc transport and desktop workspace authorization.
const mygoDraftCache = new Map();
let mygoDraftReview = null;
async function gitCall(client, action, cwd, payload = {}) {
  if (action !== 'git-create-change-request' || payload.input?.generateOnly === true) {
    return mygoOriginalGitCall(client, action, cwd, payload);
  }
  const owner = sessionOwner(state);
  const key = `${cwd}\u0000${state.gitStatus.refName || ''}`;
  let draft = mygoDraftCache.get(key);
  if (!draft || draft.client !== client) {
    draft = {client, title: '', body: '', candidate: null};
    mygoDraftCache.set(key, draft);
  }
  const input = await new Promise((resolve, reject) => {
    mygoDraftReview = {owner, cwd, client, draft, resolve, reject, loading:false, error:''};
    state.gitDialog = 'mygo-change-request';
    renderDialog();
  });
  if (!owner.owns() || state.cwd !== cwd) throw new Error('工作区已切换，请重新确认草稿');
  return mygoOriginalGitCall(client, action, cwd, {input});
}
function mygoFinishDraft(confirmed) {
  const review = mygoDraftReview;
  if (!review) return;
  mygoDraftReview = null;
  state.gitDialog = '';
  renderDialog();
  if (review.draftOnly) return;
  if (confirmed) review.resolve({title:review.draft.title.trim(), body:review.draft.body, preserveProvided:true});
  else review.reject(new Error('已取消创建；标题和正文草稿已保留'));
}
async function mygoGenerateDraft(review) {
  review.loading = true; review.error = ''; renderDialog();
  try {
    const result = await mygoOriginalGitCall(review.client, 'git-create-change-request', review.cwd, {input:{generateOnly:true}});
    if (mygoDraftReview !== review || !review.owner.owns()) return;
    review.draft.candidate = result;
  } catch (error) {
    if (mygoDraftReview === review) review.error = error.message || '生成失败，可重试或手动编辑';
  } finally {
    review.loading = false;
    if (mygoDraftReview === review) renderDialog();
  }
}
const mygoOriginalRenderDialog = renderDialog;
renderDialog = function () {
  const review = mygoDraftReview;
  if (!review) return mygoOriginalRenderDialog();
  if (!review.owner.owns() || state.cwd !== review.cwd) { mygoFinishDraft(false); return; }
  preserveSurfaceInput(dialogRoot);
  dialogRoot.replaceChildren();
  const {layer, dialog} = dialogLayer(false);
  const label = state.gitStatus.sourceControlProvider?.kind === 'gitlab' ? 'MR' : 'PR';
  dialogHead(dialog, `${label} 标题与正文`, review.draftOnly ? '生成、编辑并保留草稿，不会提交、推送或创建请求。' : '生成只准备草稿。编辑并确认后才会创建；已完成的提交或推送会保留。');
  const body = document.createElement('div'); body.className = 'dialog-body';
  body.append(descNode(`工作区：${review.cwd}\n分支：${state.gitStatus.refName || ''}`, 'row-desc'));
  body.append(descNode('标题', 'field-label'));
  const title = fieldInput(review.draft.title, `${label} 标题`, value => {review.draft.title = value; confirm.disabled = review.loading || !value.trim();});
  body.append(title);
  body.append(descNode('正文（Markdown）', 'field-label'));
  const content = document.createElement('textarea'); content.className = 'field-input'; content.rows = 9;
  content.value = review.draft.body; content.placeholder = '描述本次分支更改';
  content.addEventListener('input', () => {review.draft.body = content.value;}); body.append(content);
  const generate = ghostButton(review.loading ? '正在生成…' : '生成 / 重新生成标题与正文', () => {void mygoGenerateDraft(review);});
  generate.disabled = review.loading; body.append(generate);
  if (review.error) body.append(descNode(review.error, 'error'));
  const candidate = review.draft.candidate;
  if (candidate) {
    body.append(descNode(`生成草稿 · ${candidate.source === 'model' ? '模型' : '提交摘要'} · ${candidate.base || ''} ← ${candidate.head || ''}`, 'field-label'));
    const preview = document.createElement('pre'); preview.style.whiteSpace = 'pre-wrap'; preview.textContent = `${candidate.title}\n\n${candidate.body}`; body.append(preview);
    body.append(ghostButton('应用生成草稿到编辑框', () => {review.draft.title = candidate.title; review.draft.body = candidate.body; review.draft.candidate = null; renderDialog();}));
  }
  dialog.append(body);
  const confirm = primaryButton(review.draftOnly ? '保留草稿' : `确认标题和正文并创建 ${label}`, () => mygoFinishDraft(true));
  confirm.disabled = review.loading || !review.draft.title.trim();
  dialogFoot(dialog, [ghostButton('取消并保留草稿', () => mygoFinishDraft(false)), confirm]);
  dialogRoot.append(layer);
};

function mygoOpenDraftEditor() {
  if (!state.cwd || state.gitBusy) return;
  const client = state.chisacode.client;
  const key = `${state.cwd}\u0000${state.gitStatus.refName || ''}`;
  let draft = mygoDraftCache.get(key);
  if (!draft || draft.client !== client) {
    draft = {client, title:'', body:'', candidate:null};
    mygoDraftCache.set(key, draft);
  }
  mygoDraftReview = {owner:sessionOwner(state), cwd:state.cwd, client, draft, draftOnly:true, loading:false, error:''};
  state.gitDialog = 'mygo-change-request'; renderSheet(); renderDialog();
}
