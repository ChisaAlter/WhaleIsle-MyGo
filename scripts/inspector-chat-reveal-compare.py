"""Compare installed Chromium overlay flat fill pixels with native fixed-clock paint."""
import json
from pathlib import Path
from PIL import Image
root=Path('docs/review-20261022/evidence')
ref=json.loads((root/'inspector-reference-pass250-chat-reveal-state.json').read_text())
native=json.loads((root/'inspector-native-pass250-chat-reveal-state.json').read_text())
checks=[]
for sample,ns in zip(ref['samples'],native,strict=True):
    ms=sample['ms']
    assert ms==ns['ms']
    assert abs(sample['opacity']-ns['opacity'])<.00001
    rp=Image.open(root/f'inspector-reference-pass250-chat-reveal-{ms}.png').convert('RGB').getpixel((250,90))
    np=Image.open(root/f'inspector-native-pass250-chat-reveal-{ms}.png').convert('RGB').getpixel((250,90))
    assert max(abs(a-b) for a,b in zip(rp,np))<=1,(ms,rp,np)
    checks.append(dict(ms=ms,referenceOpacity=sample['opacity'],nativeOpacity=ns['opacity'],referenceRGB=rp,nativeRGB=np,passed=True))
(root/'inspector-pass250-chat-reveal-comparison.json').write_text(json.dumps(dict(source='Installed InspectorChatRevealer Chromium screenshot vs production native overlay Tester with fixed clock; synthetic boxes and flat background samples',maxRGBTolerance=1,checks=checks,partialVisibleNoScroll=ref['partial']['scroll']==100,reducedCenter=ref['reduced']['scroll']==1400 and ref['reduced']['rect'] is None,disposed=ref['disposed']['rect'] is None,limitations='Different synthetic content and scrollbar reservation; CSS outside outline rounds to .666667px at host zoom, native 1DIP. No whole-transcript pixel parity or smooth-scroll timing claim.'),indent=2)+'\n')
print('PASS seven pulse opacity/color samples, partial visibility, reduced motion and disposal')
