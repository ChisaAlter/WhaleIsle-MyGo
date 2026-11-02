"""Decode the installed OFL WOFF2 faces; no runtime font conversion is used.

Build tool versions: fonttools==4.61.1, brotli==1.2.0.
Usage: PYTHONPATH=<temporary tooldeps> python generate.py <installed lib/styles>

The Windows MyGo text backend consumes SFNT fonts. Variable-font merging is not
supported by fontTools.merge, so instantiate the four weights used by native UI
at each original optical-size default, then combine the exact Latin subsets.
"""
from pathlib import Path
import hashlib
import json
import sys
import tempfile
from fontTools.ttLib import TTFont
from fontTools.merge import Merger
from fontTools.varLib.instancer import instantiateVariableFont

source = Path(sys.argv[1]).resolve()
out = Path(__file__).resolve().parent
manifest = {'applicationVersion': '0.3.10', 'packageVersion': '0.2.1-rc.2',
            'fonttools': '4.61.1', 'brotli': '1.2.0', 'sourceRoot': str(source),
            'sources': [], 'files': []}
with tempfile.TemporaryDirectory() as temporary:
    temporary = Path(temporary)
    for stem in ['inter-normal', 'source-serif-4-normal', 'source-serif-4-italic']:
        sources = [source / f'{stem}-{subset}.woff2' for subset in ['latin', 'latin-ext']]
        for path in sources:
            manifest['sources'].append({'name': path.name, 'sha256': hashlib.sha256(path.read_bytes()).hexdigest()})
        for weight in [400, 500, 600, 700]:
            parts = []
            for path in sources:
                font = TTFont(path)
                axes = {a.axisTag: (weight if a.axisTag == 'wght' else a.defaultValue) for a in font['fvar'].axes}
                static = instantiateVariableFont(font, axes, inplace=False, updateFontNames=True)
                static.flavor = None
                static.recalcTimestamp = False
                target = temporary / (path.stem + f'-{weight}.ttf')
                static.save(target)
                parts.append(target)
            merged = Merger().merge([str(p) for p in parts])
            merged.recalcTimestamp = False
            target = out / f'{stem}-{weight}.ttf'
            merged.save(target)
            cmap = TTFont(target).getBestCmap()
            expected = set().union(*(set(TTFont(path).getBestCmap()) for path in parts))
            if set(cmap) != expected:
                raise ValueError(f'{target.name}: glyph coverage differs from source union')
            manifest['files'].append({'sources': [p.name for p in sources], 'output': target.name,
                                      'outputSHA256': hashlib.sha256(target.read_bytes()).hexdigest(),
                                      'weight': weight, 'axes': axes, 'glyphCoverage': len(cmap),
                                      'coverageVerified': True, 'size': target.stat().st_size})
for name in ['Inter-OFL.txt', 'SourceSerif4-OFL.txt']:
    (out / name).write_bytes((source / name).read_bytes())
(out / 'sources.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
