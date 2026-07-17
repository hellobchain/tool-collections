import os
import shutil
from pathlib import Path
from typing import Dict, Any, Optional
import zipfile
import io
import tempfile

from lxml import etree

_HAS_SOFFICE = shutil.which('soffice') is not None


NSMAP = {
    'w': 'http://schemas.openxmlformats.org/wordprocessingml/2006/main',
    'r': 'http://schemas.openxmlformats.org/officeDocument/2006/relationships',
    'wp': 'http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing',
}

def _qn(tag: str) -> str:
    prefix, local = tag.split(':')
    return f'{{{NSMAP[prefix]}}}{local}'


class DoclingConverter:
    def __init__(self):
        from docling.document_converter import DocumentConverter, PdfFormatOption
        from docling.datamodel.base_models import InputFormat
        from docling.datamodel.pipeline_options import PdfPipelineOptions

        pipeline_options = PdfPipelineOptions()
        pipeline_options.do_ocr = True
        pipeline_options.do_table_structure = True

        self.converter = DocumentConverter(
            format_options={
                InputFormat.PDF: PdfFormatOption(pipeline_options=pipeline_options)
            }
        )

    def _resolve_docx_numbering(self, docx_path: Path) -> Path:
        """Resolve auto-numbering by editing XML inside docx zip directly.
        Returns path to modified file (could be same as input if no changes)."""

        with zipfile.ZipFile(str(docx_path), 'r') as z:
            doc_xml = z.read('word/document.xml')
            numbering_xml = z.read('word/numbering.xml') if 'word/numbering.xml' in z.namelist() else None
            styles_xml = z.read('word/styles.xml') if 'word/styles.xml' in z.namelist() else None

        doc_tree = etree.fromstring(doc_xml)
        body = doc_tree.find(_qn('w:body'))
        if body is None:
            return docx_path

        # Parse numbering
        abstract_num_map: Dict[str, Dict[str, Dict]] = {}
        num_map: Dict[str, str] = {}

        if numbering_xml is not None:
            num_tree = etree.fromstring(numbering_xml)
            for abstract_num in num_tree.findall(_qn('w:abstractNum')):
                aid = abstract_num.get(_qn('w:abstractNumId'))
                levels = {}
                for lvl in abstract_num.findall(_qn('w:lvl')):
                    ilvl = lvl.get(_qn('w:ilvl'), '0')
                    nf = lvl.find(_qn('w:numFmt'))
                    lt = lvl.find(_qn('w:lvlText'))
                    st = lvl.find(_qn('w:start'))
                    levels[ilvl] = {
                        'numFmt': nf.get(_qn('w:val')) if nf is not None else 'decimal',
                        'lvlText': lt.get(_qn('w:val')) if lt is not None else '%1.',
                        'start': int(st.get(_qn('w:val'), '1')) if st is not None else 1,
                    }
                abstract_num_map[aid] = levels

            for num in num_tree.findall(_qn('w:num')):
                nid = num.get(_qn('w:numId'))
                aid_el = num.find(_qn('w:abstractNumId'))
                if aid_el is not None:
                    num_map[nid] = aid_el.get(_qn('w:val'))

            for num in num_tree.findall(_qn('w:num')):
                nid = num.get(_qn('w:numId'))
                for override in num.findall(_qn('w:lvlOverride')):
                    ilvl = override.get(_qn('w:ilvl'))
                    start_override = override.find(_qn('w:startOverride'))
                    if ilvl is not None and start_override is not None:
                        abstract_num_id = num_map.get(nid)
                        if abstract_num_id and abstract_num_id in abstract_num_map:
                            if ilvl not in abstract_num_map[abstract_num_id]:
                                abstract_num_map[abstract_num_id][ilvl] = {}
                            abstract_num_map[abstract_num_id][ilvl]['start'] = int(start_override.get(_qn('w:val'), '1'))

        # Parse styles for list style definitions
        style_defs: Dict[str, tuple] = {}
        if styles_xml is not None:
            styles_tree = etree.fromstring(styles_xml)
            for style in styles_tree.findall(_qn('w:style')):
                pPr = style.find(_qn('w:pPr'))
                if pPr is not None:
                    numPr = pPr.find(_qn('w:numPr'))
                    if numPr is not None:
                        sid = style.get(_qn('w:styleId'))
                        nid = numPr.find(_qn('w:numId'))
                        ilvl_el = numPr.find(_qn('w:ilvl'))
                        if nid is not None:
                            style_defs[sid] = (nid.get(_qn('w:val')), ilvl_el.get(_qn('w:val'), '0') if ilvl_el is not None else '0')

        # ---- text extraction helpers ----
        def _get_para_text(para_elem) -> str:
            texts: list[str] = []
            for t in para_elem.iter(_qn('w:t')):
                if t.text:
                    texts.append(t.text)
            return ''.join(texts)

        def _get_number_str(aid: Optional[str], ilvl: str, num_fmt: str, value: int) -> str:
            if aid is None:
                return ''
            if num_fmt == 'decimal':
                return str(value)
            elif num_fmt == 'lowerLetter':
                return chr(ord('a') + value - 1) if 1 <= value <= 26 else str(value)
            elif num_fmt == 'upperLetter':
                return chr(ord('A') + value - 1) if 1 <= value <= 26 else str(value)
            elif num_fmt == 'lowerRoman':
                romans = ['i', 'ii', 'iii', 'iv', 'v', 'vi', 'vii', 'viii', 'ix', 'x']
                return romans[value - 1] if 0 < value <= len(romans) else str(value)
            elif num_fmt == 'upperRoman':
                romans = ['I', 'II', 'III', 'IV', 'V', 'VI', 'VII', 'VIII', 'IX', 'X']
                return romans[value - 1] if 0 < value <= len(romans) else str(value)
            elif num_fmt == 'bullet':
                return ''
            return str(value)

        def _format_number(value: int, fmt: str, lvl_text: str, aid: str, ilvl_str: str, level_defs: Dict, counters: Dict) -> str:
            results = {}
            ilvl_int = int(ilvl_str)
            results['%1'] = _get_number_str(aid, ilvl_str, fmt, value)
            for level_offset in range(2, min(10, ilvl_int + 2)):
                parent_ilvl = ilvl_int - (level_offset - 1)
                parent_key = f'{aid}:{parent_ilvl}'
                parent_value = counters.get(parent_key, 0)
                parent_level = level_defs.get(str(parent_ilvl), {})
                parent_fmt = parent_level.get('numFmt', 'decimal') if isinstance(parent_level, dict) else 'decimal'
                pct = f'%{level_offset}'
                results[pct] = _get_number_str(aid, str(parent_ilvl), parent_fmt, parent_value)
            result = lvl_text
            for pct, val in results.items():
                result = result.replace(pct, val)
            return result

        def _find_effective_num_pr(para_elem, pPr):
            """Get effective (num_id, ilvl) for a paragraph, checking inline and pStyle."""
            numPr = pPr.find(_qn('w:numPr'))
            if numPr is not None:
                nid_el = numPr.find(_qn('w:numId'))
                ilvl_el = numPr.find(_qn('w:ilvl'))
                if nid_el is not None:
                    return nid_el.get(_qn('w:val')), ilvl_el.get(_qn('w:val')) if ilvl_el is not None else '0', numPr
            pStyle = pPr.find(_qn('w:pStyle'))
            if pStyle is not None:
                sid = pStyle.get(_qn('w:val'))
                if sid in style_defs:
                    nid, ilvl = style_defs[sid]
                    return nid, ilvl, None
            return None, '0', None

        def _is_list_style(sid: str) -> bool:
            if styles_xml is None:
                return False
            # Cache check
            if sid in _list_style_cache:
                return _list_style_cache[sid]
            styles_tree = etree.fromstring(styles_xml) if isinstance(styles_xml, bytes) else styles_xml
            for style in styles_tree.findall(_qn('w:style')):
                if style.get(_qn('w:styleId')) == sid:
                    sPPr = style.find(_qn('w:pPr'))
                    result = sPPr is not None and sPPr.find(_qn('w:numPr')) is not None
                    _list_style_cache[sid] = result
                    return result
            return False

        _list_style_cache: Dict[str, bool] = {}

        # ---- main loop ----
        counters: Dict[str, int] = {}
        prev_key: Optional[str] = None
        modified = False

        for para in body.iter(_qn('w:p')):
            if not _get_para_text(para).strip():
                continue

            pPr = para.find(_qn('w:pPr'))
            if pPr is None:
                continue

            num_id, ilvl, numPr = _find_effective_num_pr(para, pPr)
            if num_id is None:
                continue

            abstract_num_id = num_map.get(num_id)
            if abstract_num_id is None or abstract_num_id not in abstract_num_map:
                continue

            levels = abstract_num_map[abstract_num_id]
            level_def = levels.get(ilvl, {})
            if not isinstance(level_def, dict):
                continue

            num_fmt = level_def.get('numFmt', 'decimal')
            lvl_text = level_def.get('lvlText', '%1.')
            if num_fmt == 'bullet':
                continue

            # Counter logic
            if prev_key is not None:
                prev_parts = prev_key.split(':')
                prev_abstract = prev_parts[0]
                prev_ilvl = int(prev_parts[1])
            else:
                prev_abstract = None
                prev_ilvl = 0

            ilvl_int = int(ilvl)

            if prev_abstract == abstract_num_id:
                if ilvl_int <= prev_ilvl:
                    key = f'{abstract_num_id}:{ilvl}'
                    counters[key] = counters.get(key, level_def.get('start', 1) - 1) + 1
                    for k in list(counters.keys()):
                        if k.startswith(f'{abstract_num_id}:') and int(k.split(':')[1]) > ilvl_int:
                            del counters[k]
                else:
                    key = f'{abstract_num_id}:{ilvl}'
                    if key not in counters:
                        counters[key] = level_def.get('start', 1)
                    else:
                        counters[key] += 1
            else:
                key = f'{abstract_num_id}:{ilvl}'
                if key in counters:
                    counters[key] += 1
                else:
                    counters[key] = level_def.get('start', 1)

            prev_key = key
            number = counters[key]
            numbered_text = _format_number(number, num_fmt, lvl_text, abstract_num_id, ilvl, levels, counters)
            if not numbered_text:
                continue

            # Prepend numbering text
            existing_text = _get_para_text(para)
            if not existing_text:
                continue

            # Remove all existing runs (w:r) from the paragraph
            for r in para.findall(_qn('w:r')):
                para.remove(r)

            # Add a new run with numbering + text
            new_run = etree.SubElement(para, _qn('w:r'))
            new_t = etree.SubElement(new_run, _qn('w:t'))
            new_t.set('{http://www.w3.org/XML/1998/namespace}space', 'preserve')
            new_t.text = f'{numbered_text} {existing_text}'.strip()

            # Remove numPr
            if numPr is not None:
                pPr.remove(numPr)
            else:
                # Remove via pStyle
                pStyle = pPr.find(_qn('w:pStyle'))
                if pStyle is not None and _is_list_style(pStyle.get(_qn('w:val'))):
                    pPr.remove(pStyle)

            modified = True

        if not modified:
            return docx_path

        # Write modified document.xml back into the zip
        suffix = Path(docx_path).suffix
        tmp = tempfile.NamedTemporaryFile(delete=False, suffix=suffix)
        tmp_path = Path(tmp.name)

        with zipfile.ZipFile(str(docx_path), 'r') as zin:
            with zipfile.ZipFile(str(tmp_path), 'w', zipfile.ZIP_DEFLATED) as zout:
                for item in zin.infolist():
                    data = zin.read(item.filename)
                    if item.filename == 'word/document.xml':
                        data = etree.tostring(doc_tree, xml_declaration=True, encoding='UTF-8', standalone=True)
                    zout.writestr(item, data)

        tmp.close()
        return tmp_path

    def _convert_doc_to_docx(self, doc_path: Path) -> Path:
        """Convert binary .doc to .docx using LibreOffice (soffice)."""
        if not _HAS_SOFFICE:
            raise RuntimeError(
                "Cannot convert .doc file: LibreOffice not found. "
                "Install it in your Docker image: libreoffice-writer"
            )
        docx_path = doc_path.with_suffix('.docx')
        import subprocess
        subprocess.run(
            ['soffice', '--headless', '--convert-to', 'docx', '--outdir', str(doc_path.parent), str(doc_path)],
            check=True, capture_output=True, timeout=120,
        )
        if not docx_path.exists():
            raise RuntimeError(f"LibreOffice conversion failed: output not found at {docx_path}")
        return docx_path

    def convert(self, file_content: bytes, filename: str, params: Dict[str, Any]) -> Dict[str, str]:
        suffix = Path(filename).suffix.lower()
        with tempfile.NamedTemporaryFile(delete=False, suffix=suffix) as tmp:
            tmp.write(file_content)
            tmp_path = Path(tmp.name)

        cleanup_paths = [tmp_path]

        def _do_convert(path: Path) -> Any:
            if path.suffix.lower() == '.docx':
                processed = self._resolve_docx_numbering(path)
                if processed != path:
                    cleanup_paths.append(processed)
                    return self.converter.convert(processed)
            return self.converter.convert(path)

        try:
            if suffix == '.doc':
                if not _HAS_SOFFICE:
                    raise RuntimeError("Cannot convert .doc file: LibreOffice (soffice) not available")
                docx_path = self._convert_doc_to_docx(tmp_path)
                cleanup_paths.append(docx_path)
                tmp_path = docx_path

            result = _do_convert(tmp_path)

            output = {}
            to_formats = params.get('to_formats', ['md'])

            format_map = {
                'md': result.document.export_to_markdown,
                'json': lambda: result.document.export_to_dict(),
                'html': result.document.export_to_html,
                'text': result.document.export_to_text
            }

            for fmt in to_formats:
                if fmt in format_map:
                    content = format_map[fmt]()
                    if fmt == 'json':
                        import json
                        content = json.dumps(content, indent=2)
                    output[fmt] = content

            return output

        finally:
            for p in cleanup_paths:
                if p.exists():
                    p.unlink()
