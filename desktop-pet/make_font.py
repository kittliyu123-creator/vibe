"""Build a small original text atlas; no runtime font dependency."""
from pathlib import Path
import json
import re
from PIL import Image, ImageDraw, ImageFont

root = Path(__file__).resolve().parent
texts = ''.join(p.read_text() for p in root.glob('*.go'))
chars = sorted(set(re.findall(r'[\u3000-\u303f\u3400-\u9fff\uff00-\uffef]', texts)) | set(' ?!0123456789'))
font = ImageFont.truetype('/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc', 19, index=2)
atlas = Image.new('RGBA', (32 * 24, ((len(chars)+31)//32)*28))
draw = ImageDraw.Draw(atlas)
for i, char in enumerate(chars):
    draw.text(((i % 32)*24, (i//32)*28+1), char, font=font, fill=(66, 60, 76, 255), anchor='lt')
atlas.save(root/'assets/font.png')
(root/'assets/glyphs.json').write_text(json.dumps({c:i for i,c in enumerate(chars)}, ensure_ascii=False), encoding='utf-8')
print(f'Generated {len(chars)} glyphs')
