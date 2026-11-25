from pathlib import Path
import json,re,html
from reportlab.lib import colors
from reportlab.lib.enums import TA_CENTER
from reportlab.lib.styles import getSampleStyleSheet,ParagraphStyle
from reportlab.lib.pagesizes import A4
from reportlab.platypus import SimpleDocTemplate,Paragraph,Spacer,PageBreak,KeepTogether
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
root=Path(__file__).resolve().parents[1]
out=root/'docs'/'BalanzaPro3.0-Master-Blueprint.pdf'
font=Path('/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf')
if font.exists():
 pdfmetrics.registerFont(TTFont('DV',str(font)))
 pdfmetrics.registerFont(TTFont('DV-Bold',str(font.with_name('DejaVuSans-Bold.ttf'))))
 normal,bold='DV','DV-Bold'
else: normal,bold='Helvetica','Helvetica-Bold'
navy=colors.HexColor('#18354A'); teal=colors.HexColor('#126B69'); grey=colors.HexColor('#52616E')
styles={
 'cover':ParagraphStyle('cover',fontName=bold,fontSize=31,leading=37,textColor=navy,spaceAfter=18),
 'sub':ParagraphStyle('sub',fontName=normal,fontSize=14,leading=20,textColor=grey,spaceAfter=16),
 'h1':ParagraphStyle('h1',fontName=bold,fontSize=20,leading=26,textColor=navy,spaceAfter=12,keepWithNext=True),
 'h2':ParagraphStyle('h2',fontName=bold,fontSize=13,leading=18,textColor=teal,spaceBefore=10,spaceAfter=6,keepWithNext=True),
 'body':ParagraphStyle('body',fontName=normal,fontSize=9.5,leading=15,textColor=navy,spaceAfter=7,splitLongWords=True),
 'small':ParagraphStyle('small',fontName=normal,fontSize=8,leading=12,textColor=grey,spaceAfter=5,splitLongWords=True),
 'adr':ParagraphStyle('adr',fontName=bold,fontSize=9.5,leading=13,textColor=teal,spaceBefore=5,spaceAfter=3,keepWithNext=True)
}
def clean(s):
 s=re.sub(r'\[([^\]]+)\]\(([^)]+)\)',r'\1 (\2)',s)
 return html.escape(s.replace('**','').replace('`',''))
def para(s,style='body'): return Paragraph(clean(s),styles[style])
story=[Spacer(1,95),para('BALANZAPRO3.0','cover'),para('Master Blueprint','h1'),para('Arquitectura · reglas comerciales · operaciones · desarrollo con IA','sub'),Spacer(1,20),para('DOCUMENTACIÓN BASE / v0.1 / 25 NOV 2026','h2'),para('168 decisiones de diseño · 13 skills · Plan incremental'),para('Preparado para implementación por etapas. No constituye un producto desplegado ni una certificación fiscal.','sub'),Spacer(1,24),para('Fuente de verdad: archivos Markdown y registro ADR del repositorio.','small')]
docs=['docs/MASTER.md','docs/architecture/overview.md','docs/security/security.md','docs/design-system/UX.md',*sorted(str(p.relative_to(root)) for p in (root/'docs/business-rules').glob('*.md')),'docs/deployment/operations.md','docs/runbooks/incident-response.md','docs/testing/STRATEGY.md','docs/architecture/agents.md','ROADMAP.md','BACKLOG.md','TECH_STACK.md','AGENTS.md','PROMPT_INICIAL.md']
for f in docs:
 story.append(PageBreak())
 for line in (root/f).read_text().splitlines():
  line=line.strip()
  if not line: continue
  if line.startswith('# '): story.append(para(line[2:],'h1'))
  elif line.startswith('## '): story.append(para(line[3:],'h2'))
  elif line.startswith('|'):
   if re.fullmatch(r'[| :\-]+',line): continue
   story.append(para('  /  '.join(s.strip() for s in line.strip('|').split('|')),'small'))
  else: story.append(para(line))
story.extend([PageBreak(),para('Anexo | Registro ADR 001–168','h1'),para('Aprobación de diseño, no implementación. Las decisiones 001–033 se reconstruyeron del contexto disponible. Revisar propuestas pendientes y solapamientos antes de programar.')])
for r in json.loads((root/'docs/decisions/registry.json').read_text()):
 story.append(para('ADR-'+r['id']+' · '+r['title']+' · '+r['status'],'adr'))
 story.append(para(r['decision']))
def footer(c,d):
 w,h=A4
 c.setStrokeColor(colors.HexColor('#D6E0E4'));c.line(45,42,w-45,42)
 c.setFont(normal,8);c.setFillColor(grey)
 c.drawString(45,27,'BalanzaPro3.0  |  Master Blueprint  |  v0.1')
 c.drawRightString(w-45,27,str(d.page))
SimpleDocTemplate(str(out),pagesize=A4,rightMargin=46,leftMargin=46,topMargin=46,bottomMargin=56,title='BalanzaPro3.0 - Master Blueprint',author='BalanzaPro3.0').build(story,onFirstPage=footer,onLaterPages=footer)
print(out)
