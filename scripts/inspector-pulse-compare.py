"""Compare real browser WAAPI samples with native fixed-clock production row paint."""
import json
from pathlib import Path
from PIL import Image
root=Path('docs/review-20261022/evidence')
reference=json.loads((root/'inspector-reference-pass249-pulse-state.json').read_text())
checks=[]
for ms in (0,350,700):
    ref=Image.open(root/f'inspector-reference-pass249-pulse-{ms}.png').convert('RGB').getpixel((700,65))
    for mode in ('session-log','chat-node'):
        native=Image.open(root/f'inspector-native-pass249-pulse-{mode}-{ms}.png').convert('RGB').getpixel((700,5))
        if max(abs(a-b) for a,b in zip(ref,native))>1:
            raise AssertionError((ms,mode,ref,native))
        checks.append(dict(ms=ms,mode=mode,referenceRGB=ref,nativeRGB=native,passed=True))
for mode in ('session-log','chat-node'):
    pixel=Image.open(root/f'inspector-native-pass249-pulse-{mode}-reduced.png').convert('RGB').getpixel((700,5))
    assert pixel==(255,255,255)
assert reference['initialAnimations']==0 and reference['rawUpdateAnimations']==0 and reference['rawRevealAnimations']==1 and reference['reducedRevealAnimations']==0
(root/'inspector-pass249-pulse-comparison.json').write_text(json.dumps(dict(source='Installed InspectorTable WAAPI real browser screenshot vs native Tester production row painter with fixed clock; excludes typography',maxRGBTolerance=1,checks=checks,reducedMotionWhite=True),indent=2)+'\n')
print('PASS six cross-renderer pulse samples; reduced motion silent')
