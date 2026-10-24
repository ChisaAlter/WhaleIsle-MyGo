"""Reconcile retained installed-browser measurements and native WebBlock geometry."""
from pathlib import Path
import hashlib
import json

ROOT = Path(__file__).resolve().parent.parent
DIR = ROOT / 'docs/review-20261022/evidence'
RUN = 'tool-web-pass265-reference-2026-10-24T04-14-24-538Z-4fd967fc'
reference_path = DIR / (RUN + '.json')
source_path = DIR / (RUN + '-source.json')
reference = json.loads(reference_path.read_text(encoding='utf-8'))
source = json.loads(source_path.read_text(encoding='utf-8'))
native = {row['case']: row['rects'] for row in json.loads((DIR / 'tool-web-pass265-native-preset-geometry.json').read_text(encoding='utf-8'))}
early = reference['measurements']['early']
light = early['light']
rows = light['short']['rows']
checks = []
def check(name, actual, expected, tolerance=.04):
    checks.append({'name': name, 'actual': actual, 'expected': expected, 'tolerance': tolerance,
                   'pass': abs(actual-expected) <= tolerance})
check('ordinary source row height',native['source-only']['card']['H']-24,early['actualNaturalTitleOnlySourceRowHeight'])
check('full source row height',native['source-full']['card']['H']-24,rows[0]['li']['box']['h'])
check('two-source truncated card height',native['two-sources']['card']['H'],light['short']['card']['box']['h'])
check('two-source card width',native['two-sources']['card']['W'],light['short']['card']['box']['w'])
check('source list height',native['two-sources']['card']['H']-24-28,light['short']['sources']['box']['h'])
check('source indentation',native['two-sources']['Source title']['X']-14,rows[0]['li']['box']['x']-light['short']['sources']['box']['x'])
check('full source snippet top',native['source-full']['Snippet text']['Y']-12,rows[0]['snippet']['box']['y']-rows[0]['li']['box']['y'])
check('full source snippet height',native['source-full']['Snippet text']['H'],rows[0]['snippet']['box']['h'])
check('full source date top',native['source-full']['2026-10-24']['Y']-12,rows[0]['published']['box']['y']-rows[0]['li']['box']['y'])
check('full source date height',native['source-full']['2026-10-24']['H'],rows[0]['published']['box']['h'])
check('second source top',native['two-sources']['Second title']['Y']-12,rows[1]['li']['box']['y']-rows[0]['li']['box']['y'])
check('truncation label top',native['two-sources']['Source list truncated']['Y']-12,light['short']['truncated']['box']['y']-rows[0]['li']['box']['y'])
check('fetch card height',native['fetch-truncated']['card']['H'],light['fetch']['card']['box']['h'])
check('fetch URL height',native['fetch-truncated']['https://final.example.test/b']['H'],light['fetch']['fetchUrl']['box']['h'])
check('fetch status top',native['fetch-truncated']['HTTP 200']['Y']-12,light['fetch']['status']['box']['y']-light['fetch']['fetchUrl']['box']['y'])
check('fetch metadata height',native['fetch-truncated']['HTTP 200']['H'],light['fetch']['status']['box']['h'])
check('fetch metadata gap',native['fetch-truncated']['Content truncated']['X']-native['fetch-truncated']['HTTP 200']['X']-native['fetch-truncated']['HTTP 200']['W'],12)
for case, key in [('empty','empty'),('empty-truncated','emptyAnswer'),('whitespace-answer','space'),('answer-only','answer')]:
    check(case+' height',native[case]['card']['H'],reference['measurements']['presence'][key]['card']['h'])
last=reference['measurements']['fallbacks']['geometry']['rows'][-1]
check('collapsed empty snippet/date block margins',native['source-space-details']['card']['H']-24,last['li']['box']['h'])
for row in native.values():
    if '1.' in row:
        check('ordered marker gutter width '+str(row['card']['H']),row['1.']['W'],35)

hashes=[]
for file in source['files']:
    actual=hashlib.sha256(Path(file['path']).read_bytes()).hexdigest()
    hashes.append({'id':file['id'],'path':file['path'],'sha256':actual,'pass':actual==file['sha256']})

result={'schema':'tool-web-pass265-reconciled/v1','browserStatus':reference['status'],
        'browserSummary':reference['summary'],'browserFailuresRetained':reference['summary']['failedAssertions']==2,
        'geometry':{'passed':sum(c['pass'] for c in checks),'total':len(checks),'checks':checks},
        'sourceHashes':{'passed':sum(h['pass'] for h in hashes),'total':len(hashes),'files':hashes},
        'browserCleanup':reference['cleanup'],'combinedSourceMetricsBytes':reference_path.stat().st_size+source_path.stat().st_size,
        'limits':['Browser acceptance remains FAILED 8/10 cases and46/48 assertions; geometry reconciliation does not change it.',
                  'Browser fixtures use symbolic JSX/manual DOM and canonical answer DOM, not actual React/Markdown/Host/full app.',
                  'Browser fetch header globe is approximate; exact emitted caller source records Browse14.',
                  'Source anchor glyph bounds, marker glyph baselines, multiline hit fragments and link raster parity are not certified by these box comparisons.',
                  'No native scrollbar, physical input, actual browser navigation or OS clipboard certificate.']}
output=DIR/'tool-web-pass265-reconciled.json'
output.write_text(json.dumps(result,ensure_ascii=True,indent=2)+'\n',encoding='utf-8')
print('Geometry',result['geometry']['passed'],'/',len(checks),'source hashes',result['sourceHashes']['passed'],'/',len(hashes),'browser retained',reference['status'])
if not all(c['pass'] for c in checks) or not all(h['pass'] for h in hashes) or not result['browserFailuresRetained'] or not reference['cleanup']['pass']:
    raise SystemExit(1)
