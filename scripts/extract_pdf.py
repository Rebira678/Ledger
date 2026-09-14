#!/usr/bin/env python3
"""Extract text from ReportLab-generated PDFs (ASCII85Decode + FlateDecode streams)."""
import base64
import re
import sys
import zlib


def extract_streams(raw: bytes) -> list[str]:
    """Pull out all ASCII85+Flate streams and return decoded text."""
    texts = []
    # Find stream objects with ASCII85Decode + FlateDecode filters
    pattern = re.compile(
        rb"<<\s*/Filter\s*\[\s*/ASCII85Decode\s*/FlateDecode\s*\]\s*/Length\s+(\d+)\s*>>\s*stream\r?\n(.*?)endstream",
        re.DOTALL,
    )
    for m in pattern.finditer(raw):
        length = int(m.group(1))
        data = m.group(2)[:length]
        # Strip ASCII85 wrapper (~> terminator)
        if data.rstrip().endswith(b"~>"):
            data = data.rstrip()[:-2]
        try:
            decoded = base64.a85decode(data, adobe=False)
        except Exception as e:
            print(f"[warn] a85decode failed: {e}", file=sys.stderr)
            continue
        try:
            inflated = zlib.decompress(decoded)
        except Exception as e:
            print(f"[warn] zlib failed: {e}", file=sys.stderr)
            continue
        texts.append(inflated)
    return texts


def pdf_text_from_content(content: bytes) -> str:
    """Very small PDF text-operator parser: extracts strings shown by Tj/TJ."""
    out: list[str] = []
    # Tokenize strings inside parentheses (handling escapes) followed by Tj/TJ
    i = 0
    n = len(content)
    pending: list[str] = []

    def flush_tj():
        if pending:
            out.append("".join(pending))
            pending.clear()

    while i < n:
        c = content[i]
        if c == 0x28:  # (
            i += 1
            buf = bytearray()
            depth = 1
            while i < n and depth > 0:
                ch = content[i]
                if ch == 0x5C:  # backslash escape
                    i += 1
                    if i >= n:
                        break
                    e = content[i]
                    if e in b"nrtbf":
                        mapping = {ord("n"): 10, ord("r"): 13, ord("t"): 9, ord("b"): 8, ord("f"): 12}
                        buf.append(mapping[e])
                    elif e in b"()\\":
                        buf.append(e)
                    elif 0x30 <= e <= 0x37:  # octal
                        oct_digits = bytes([e])
                        i += 1
                        while i < n and len(oct_digits) < 3 and 0x30 <= content[i] <= 0x37:
                            oct_digits += bytes([content[i]])
                            i += 1
                        i -= 1
                        buf.append(int(oct_digits, 8) & 0xFF)
                    else:
                        buf.append(e)
                    i += 1
                elif ch == 0x28:  # (
                    depth += 1
                    buf.append(ch)
                    i += 1
                elif ch == 0x29:  # )
                    depth -= 1
                    if depth == 0:
                        break
                    buf.append(ch)
                    i += 1
                else:
                    buf.append(ch)
                    i += 1
            pending.append(buf.decode("cp1252", errors="replace"))
            i += 1
        elif c == 0x54 and i + 1 < n:  # T or T
            two = content[i : i + 2]
            if two == b"Tj":
                flush_tj() if False else None
                # Tj: output the pending string immediately
                if pending:
                    out.append("".join(pending))
                    pending.clear()
                out.append("\n")
                i += 2
            elif two == b"TJ":
                flush_tj()
                out.append("\n")
                i += 2
            else:
                i += 1
        else:
            i += 1
    flush_tj()
    return "".join(out)


def main() -> None:
    if len(sys.argv) < 2:
        print("usage: extract_pdf.py FILE.pdf", file=sys.stderr)
        sys.exit(1)
    path = sys.argv[1]
    with open(path, "rb") as f:
        raw = f.read()
    for idx, stream in enumerate(extract_streams(raw)):
        print(f"===== STREAM {idx} =====")
        print(pdf_text_from_content(stream))


if __name__ == "__main__":
    main()
