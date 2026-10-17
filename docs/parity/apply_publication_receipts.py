#!/usr/bin/env python3
"""Apply explicit publication/CI receipts; never infer source or runtime completion.
Usage: python apply_publication_receipts.py progress.json receipts.json --markdown summary.md
Receipt envelope: {"receipts":[{"gap_id":"WT05","stage":"source-fill-border",
"sha":"<40 hex>","url":"https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/<sha>",
"evidence_source":"GitHub API response or explicit parent confirmation",
"verified_at":"ISO timestamp"}],"ci":[{"gap_id":"CH12","stage":"inline-and-streaming",
"run_id":123,"url":"https://github.com/.../actions/runs/123","status":"in_progress",
"conclusion":null,"source":"GitHub API response"}]}.
Existing known stage names are required. Applying identical receipts is idempotent.
"""
import argparse,json,re
from pathlib import Path

def render(d):
    by={i['id']:i for i in d['work_items']}
    lines=['# 重建进度记录（2026-10-17）','',
    '依据已有审计、冻结交付和明确发布回执整理；没有新代码审查、测试或真机验收。完整记录见 [JSON](reconstruction-progress-20261017.json)。','',
    '- 历史工作项与能力索引存在重叠，不是全项目分母，不给总体完成百分比。',
    '- source-implemented、已冻结、已推送、CI 通过和真机验收分别记录。',
    '- 当前缺口只见 remaining；旧发现保留在 historical_finding / audit_states / superseded_progress。',
    '- 范围：Windows/macOS 桌面及已有 Android 远程；Linux 不纳入产品验收。','',
    '## 当前关键状态','']
    for ident in ['WT05','CP08','CH12','PR07','PR06','AX203','CH68','WT08']:
        i=by[ident];lines += [f"### {ident} · {i['status']}",'']
        for v in i.get('implemented',[]): lines.append('- 已有源码：'+v)
        for v in i.get('remaining',[]): lines.append('- 当前剩余：'+v)
        for s in i.get('delivery_stages',[]):
            shas=[v if isinstance(v,str) else v.get('sha','') for v in s.get('publication_commits',[])]
            lines.append('- 阶段 '+s['stage']+'：'+s.get('publication_status','unreconciled')+('；'+', '.join(shas) if shas else '；SHA 待实证补齐'))
        lines.append('')
    lines += ['## 真实桌面验收','',
    '所有相关 work item 的 runtime_status 仍为 runtime-unverified。原生窗口、输入/裁切、Finder/Explorer 拖放、实际 Host 模块可用性、Android 远程交互等未因单元检查或 CI 成功而获得真机验收。CH12 stage 1 成功 CI 不代表 stage 2；两阶段 CI 分别记录。','',
    '## 发布回执补齐','',
    '发布状态只依据完整提交号和已核实的远端记录。源码交付、提交、原生编译和真实运行验收分别记录；缺少独立提交证据的条目保留空值，不以猜测补齐。','',
    '## 已记录工作项索引','',
    '以下列出本次汇总的全部工作项。它们包含拆分项与历史范围外项，不能用条数计算全项目完成率；详细证据、剩余差异及运行状态以 JSON 为准。','',
    '| ID | 工作项 | 源码状态 | 已核实提交 |','|---|---|---|---|']
    labels={'source-implemented':'有实现证据','partial':'部分实现','blocked':'受阻','runtime-unverified':'证据不足','historical-out-of-scope':'历史范围外','withdrawn':'不适用'}
    for i in d['work_items']:
        sha=i.get('publication_commit')
        commit=f'[{sha[:8]}](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/{sha})' if sha else '未核对独立提交'
        title=str(i.get('title','')).replace('|','/').replace('\n',' ')
        lines.append(f"| {i['id']} | {title} | {labels.get(i['status'],i['status'])} | {commit} |")
    lines += ['', '## 本轮发布与验证边界', '',
        '- 已核实开发分支头：`'+d.get('reconciliation',{}).get('verified_head','unknown')+'`；提交条数仅为发布记录数。',
        '- 所有真实平台验收仍单列为 unknown，不由源码、冻结或 CI 结果推定。']
    for i in d['work_items']:
        b=i.get('verification_boundary',{})
        for path in b.get('unpublished_test_files',[]): lines.append('- '+i['id']+' 尚未发布的测试文件：`'+path+'`。')
        if b.get('api_and_image_native_sdk_compile')=='not-run': lines.append('- '+i['id']+' API/图片增量尚未做原生 SDK 编译；此前 inline/streaming 成功 CI 不覆盖该增量。')
    lines += ['- OpenUI 完整运行包许可仍未明确，见 [许可证据](../openui-runtime-license-status-20261017.md)；地图仅有架构边界核实，尚无能力实现。', '']
    return '\n'.join(lines)

def apply(d,r):
    by={i['id']:i for i in d['work_items']}
    for receipt in r.get('receipts',[]):
        i=by[receipt['gap_id']]
        s=next(x for x in i.get('delivery_stages',[]) if x['stage']==receipt['stage'])
        sha=receipt['sha'];url=receipt['url']
        if not re.fullmatch('[0-9a-f]{40}',sha):raise ValueError('Full lowercase SHA required')
        if url!='https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/'+sha:raise ValueError('Exact repository commit URL required')
        if not receipt.get('evidence_source') or not receipt.get('verified_at'):raise ValueError('Receipt source/time required')
        known=s.setdefault('publication_commits',[])
        if sha not in [v if isinstance(v,str) else v.get('sha') for v in known]:known.append(sha)
        s['publication_status']='verified-pushed'
        if receipt not in s.setdefault('publication_receipts',[]):s['publication_receipts'].append(receipt)
        known=i.setdefault('publication_commits',[])
        if sha not in [v if isinstance(v,str) else v.get('sha') for v in known]:known.append({'sha':sha,'url':url,'evidence_source':receipt['evidence_source'],'verified_at':receipt['verified_at']})
        i['publication_commit']=sha
        stages=i.get('delivery_stages',[])
        i['publication_status']='verified-pushed' if all(x.get('publication_status')=='verified-pushed' for x in stages) else 'partially-verified-stages-pending'
    for receipt in r.get('ci',[]):
        i=by[receipt['gap_id']]
        s=next(x for x in i.get('delivery_stages',[]) if x['stage']==receipt['stage'])
        if not receipt.get('source') or not receipt.get('url') or not receipt.get('run_id'):raise ValueError('CI evidence source/run required')
        s['ci']=receipt
    return d

if __name__=='__main__':
    parser=argparse.ArgumentParser();parser.add_argument('progress');parser.add_argument('receipts');parser.add_argument('--markdown')
    args=parser.parse_args();p=Path(args.progress)
    d=apply(json.loads(p.read_text()),json.loads(Path(args.receipts).read_text()))
    p.write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n')
    if args.markdown:Path(args.markdown).write_text(render(d))
