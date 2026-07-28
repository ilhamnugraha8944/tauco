import {
  existsSync,
  readFileSync,
  rmSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { mkdtemp, realpath } from "node:fs/promises";
import { tmpdir } from "node:os";
import { basename, dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import { chromium } from "@playwright/test";

const projectRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const markdownPath = join(projectRoot, "plan.md");
const pdfPath = join(projectRoot, "plan.pdf");

function escapeHtml(value) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function renderInline(value) {
  const codeSpans = [];
  let rendered = escapeHtml(value).replace(/`([^`]+)`/g, (_, code) => {
    const token = `%%CODE_SPAN_${codeSpans.length}%%`;
    codeSpans.push(`<code>${code}</code>`);
    return token;
  });

  rendered = rendered
    .replace(
      /\[([^\]]+)\]\(([^)\s]+)\)/g,
      '<a href="$2">$1</a>',
    )
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/(?<!\*)\*([^*]+)\*(?!\*)/g, "<em>$1</em>");

  for (const [index, code] of codeSpans.entries()) {
    rendered = rendered.replace(`%%CODE_SPAN_${index}%%`, code);
  }

  return rendered;
}

function splitTableRow(line) {
  return line
    .trim()
    .replace(/^\|/, "")
    .replace(/\|$/, "")
    .split("|")
    .map((cell) => cell.trim());
}

function isTableDivider(line) {
  return /^\s*\|?\s*:?-{3,}:?\s*(\|\s*:?-{3,}:?\s*)+\|?\s*$/.test(
    line,
  );
}

function isBlockStart(lines, index) {
  const line = lines[index] ?? "";
  const nextLine = lines[index + 1] ?? "";

  return (
    /^#{1,6}\s+/.test(line) ||
    /^```/.test(line) ||
    /^>\s?/.test(line) ||
    /^(\s*)[-*+]\s+/.test(line) ||
    /^\s*\d+\.\s+/.test(line) ||
    /^-{3,}\s*$/.test(line) ||
    (line.includes("|") && isTableDivider(nextLine))
  );
}

function renderMarkdown(markdown) {
  const lines = markdown.replaceAll("\r\n", "\n").split("\n");
  const html = [];
  let paragraph = [];
  let listType = null;
  let listItems = [];

  function flushParagraph() {
    if (paragraph.length === 0) {
      return;
    }

    html.push(`<p>${renderInline(paragraph.join(" ").trim())}</p>`);
    paragraph = [];
  }

  function flushList() {
    if (!listType || listItems.length === 0) {
      listType = null;
      listItems = [];
      return;
    }

    const tag = listType === "ordered" ? "ol" : "ul";
    const items = listItems
      .map((item) => {
        const taskMatch = item.match(/^\[([ xX])\]\s+(.+)$/);

        if (!taskMatch) {
          return `<li>${renderInline(item)}</li>`;
        }

        const checked = taskMatch[1].toLowerCase() === "x";
        return [
          '<li class="task-item">',
          `<span class="checkbox${checked ? " checked" : ""}" aria-hidden="true">${checked ? "&#10003;" : ""}</span>`,
          `<span>${renderInline(taskMatch[2])}</span>`,
          "</li>",
        ].join("");
      })
      .join("");

    html.push(`<${tag}>${items}</${tag}>`);
    listType = null;
    listItems = [];
  }

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index];
    const trimmed = line.trim();

    if (!trimmed) {
      flushParagraph();
      flushList();
      continue;
    }

    if (/^```/.test(trimmed)) {
      flushParagraph();
      flushList();
      const language = trimmed.slice(3).trim();
      const code = [];
      index += 1;

      while (index < lines.length && !/^```/.test(lines[index].trim())) {
        code.push(lines[index]);
        index += 1;
      }

      html.push(
        `<pre data-language="${escapeHtml(language)}"><code>${escapeHtml(code.join("\n"))}</code></pre>`,
      );
      continue;
    }

    const headingMatch = line.match(/^(#{1,6})\s+(.+)$/);
    if (headingMatch) {
      flushParagraph();
      flushList();
      const level = headingMatch[1].length;
      const text = headingMatch[2].trim();
      const headingClass =
        level === 2 && /^G\d+\./.test(text) ? ' class="gate-title"' : "";
      html.push(
        `<h${level}${headingClass}>${renderInline(text)}</h${level}>`,
      );
      continue;
    }

    if (/^-{3,}\s*$/.test(trimmed)) {
      flushParagraph();
      flushList();
      html.push("<hr>");
      continue;
    }

    if (trimmed.startsWith(">")) {
      flushParagraph();
      flushList();
      const quote = [];

      while (index < lines.length && lines[index].trim().startsWith(">")) {
        quote.push(lines[index].trim().replace(/^>\s?/, ""));
        index += 1;
      }

      index -= 1;
      html.push(
        `<blockquote>${renderInline(quote.join(" ").trim())}</blockquote>`,
      );
      continue;
    }

    if (
      line.includes("|") &&
      index + 1 < lines.length &&
      isTableDivider(lines[index + 1])
    ) {
      flushParagraph();
      flushList();
      const headers = splitTableRow(line);
      const rows = [];
      index += 2;

      while (
        index < lines.length &&
        lines[index].trim() &&
        lines[index].includes("|")
      ) {
        rows.push(splitTableRow(lines[index]));
        index += 1;
      }

      index -= 1;
      const head = headers
        .map((cell) => `<th>${renderInline(cell)}</th>`)
        .join("");
      const body = rows
        .map(
          (row) =>
            `<tr>${headers
              .map(
                (_, cellIndex) =>
                  `<td>${renderInline(row[cellIndex] ?? "")}</td>`,
              )
              .join("")}</tr>`,
        )
        .join("");

      html.push(
        `<div class="table-wrap"><table><thead><tr>${head}</tr></thead><tbody>${body}</tbody></table></div>`,
      );
      continue;
    }

    const unorderedMatch = line.match(/^\s*[-*+]\s+(.+)$/);
    const orderedMatch = line.match(/^\s*\d+\.\s+(.+)$/);

    if (unorderedMatch || orderedMatch) {
      flushParagraph();
      const nextListType = orderedMatch ? "ordered" : "unordered";

      if (listType && listType !== nextListType) {
        flushList();
      }

      listType = nextListType;
      listItems.push((unorderedMatch ?? orderedMatch)[1].trim());
      continue;
    }

    if (
      listType &&
      /^\s{2,}\S/.test(line) &&
      listItems.length > 0
    ) {
      listItems[listItems.length - 1] += ` ${trimmed}`;
      continue;
    }

    if (listType) {
      flushList();
    }

    paragraph.push(trimmed);

    const nextIndex = index + 1;
    if (
      nextIndex >= lines.length ||
      !lines[nextIndex].trim() ||
      isBlockStart(lines, nextIndex)
    ) {
      flushParagraph();
    }
  }

  flushParagraph();
  flushList();
  return html.join("\n");
}

export function buildDocument(markdown) {
  const title =
    markdown.match(/^#\s+(.+)$/m)?.[1] ?? "Phase 1A Launch Completion Plan";
  const subtitle =
    markdown.match(/^##\s+(.+)$/m)?.[1] ?? "Tauco Cap Badak Website";
  const documentVersion =
    markdown.match(/^\|\s*Versi\s*\|\s*([^|]+?)\s*\|$/m)?.[1] ?? "1.0";
  const documentDate =
    markdown.match(/^\|\s*Tanggal\s*\|\s*([^|]+?)\s*\|$/m)?.[1] ??
    "Tanggal tidak tersedia";
  const bodyMarkdown = markdown
    .replace(/^#\s+.+\r?\n(?:\r?\n)?/, "")
    .replace(/^##\s+.+\r?\n(?:\r?\n)?/, "");
  const body = renderMarkdown(bodyMarkdown);

  return `<!doctype html>
<html lang="id-ID">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <base href="https://github.com/ilhamnugraha8944/tauco/blob/main/">
    <title>${escapeHtml(title)}</title>
    <style>
      @page {
        size: A4;
        margin: 16mm 15mm 18mm;
      }

      :root {
        --ink: #17211d;
        --muted: #5f6d66;
        --forest: #2f654e;
        --forest-deep: #173f31;
        --forest-soft: #dce9e1;
        --paper: #f6f7f4;
        --line: #bdc9c2;
        --warm: #b98a57;
      }

      * {
        box-sizing: border-box;
      }

      html {
        color: var(--ink);
        font-family: "Segoe UI", Arial, sans-serif;
        font-size: 10.2pt;
        line-height: 1.48;
        print-color-adjust: exact;
        -webkit-print-color-adjust: exact;
      }

      body {
        margin: 0;
        background: white;
      }

      .cover {
        min-height: 255mm;
        display: flex;
        flex-direction: column;
        justify-content: space-between;
        page-break-after: always;
        break-after: page;
        border: 1.2pt solid var(--forest-deep);
        background:
          radial-gradient(circle at 85% 8%, rgba(185, 138, 87, 0.2), transparent 28%),
          linear-gradient(145deg, #f7f8f5 0%, #eef3ef 62%, #dde9e1 100%);
        padding: 18mm 16mm;
        position: relative;
        overflow: hidden;
      }

      .cover::after {
        content: "";
        position: absolute;
        right: -28mm;
        bottom: -32mm;
        width: 110mm;
        height: 110mm;
        border: 18mm solid rgba(47, 101, 78, 0.08);
        border-radius: 50%;
      }

      .cover-kicker {
        color: var(--forest);
        font-size: 9pt;
        font-weight: 700;
        letter-spacing: 0.18em;
        text-transform: uppercase;
      }

      .cover-rule {
        width: 30mm;
        height: 3px;
        margin: 8mm 0 12mm;
        background: var(--warm);
      }

      .cover h1 {
        max-width: 150mm;
        margin: 0;
        color: var(--forest-deep);
        font-size: 32pt;
        line-height: 1.06;
        letter-spacing: -0.035em;
      }

      .cover h2 {
        margin: 5mm 0 0;
        color: var(--muted);
        font-size: 16pt;
        font-weight: 500;
      }

      .cover-gates {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 3mm;
        margin-top: 16mm;
        max-width: 150mm;
      }

      .cover-gates span {
        border: 1px solid rgba(47, 101, 78, 0.34);
        background: rgba(255, 255, 255, 0.64);
        color: var(--forest-deep);
        font-size: 8.2pt;
        font-weight: 600;
        padding: 2.5mm 3mm;
      }

      .cover-meta {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 3mm 10mm;
        max-width: 120mm;
        position: relative;
        z-index: 1;
      }

      .cover-meta div {
        border-top: 1px solid rgba(23, 63, 49, 0.3);
        padding-top: 2mm;
      }

      .cover-meta small {
        display: block;
        color: var(--muted);
        font-size: 7.5pt;
        letter-spacing: 0.08em;
        text-transform: uppercase;
      }

      .cover-meta strong {
        display: block;
        margin-top: 1mm;
        color: var(--forest-deep);
        font-size: 10pt;
      }

      main {
        width: 100%;
      }

      h1, h2, h3, h4 {
        color: var(--forest-deep);
        break-after: avoid;
        page-break-after: avoid;
      }

      h1 {
        margin: 0 0 7mm;
        font-size: 25pt;
        letter-spacing: -0.03em;
      }

      h2 {
        margin: 10mm 0 4mm;
        padding-bottom: 2mm;
        border-bottom: 1.3pt solid var(--forest);
        font-size: 17pt;
        letter-spacing: -0.02em;
      }

      h2.gate-title {
        break-before: page;
        page-break-before: always;
        margin-top: 0;
        padding: 4mm 5mm;
        border: 0;
        background: var(--forest-deep);
        color: white;
      }

      h3 {
        margin: 7mm 0 2.5mm;
        font-size: 12.5pt;
      }

      h4 {
        margin: 5mm 0 2mm;
        font-size: 10.8pt;
      }

      p {
        margin: 0 0 3.2mm;
        orphans: 3;
        widows: 3;
      }

      strong {
        color: #102b21;
      }

      a {
        color: var(--forest);
        text-decoration: none;
        overflow-wrap: anywhere;
      }

      blockquote {
        margin: 4mm 0 6mm;
        padding: 3.5mm 5mm;
        border-left: 3px solid var(--warm);
        background: #f5f0e9;
        color: #4e4b43;
        break-inside: avoid;
        page-break-inside: avoid;
      }

      ul, ol {
        margin: 2mm 0 4mm;
        padding-left: 6mm;
      }

      li {
        margin: 1.2mm 0;
        padding-left: 1mm;
      }

      .task-item {
        display: flex;
        align-items: flex-start;
        gap: 2.2mm;
        list-style: none;
        margin-left: -6mm;
        padding-left: 0;
        break-inside: avoid;
      }

      .checkbox {
        width: 3.6mm;
        height: 3.6mm;
        flex: 0 0 3.6mm;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        margin-top: 0.7mm;
        border: 1px solid #6d7d74;
        border-radius: 0.5mm;
        color: white;
        font-size: 7pt;
        line-height: 1;
      }

      .checkbox.checked {
        border-color: var(--forest);
        background: var(--forest);
      }

      .table-wrap {
        width: 100%;
        margin: 3mm 0 5mm;
      }

      table {
        width: 100%;
        border-collapse: collapse;
        table-layout: auto;
        font-size: 8.6pt;
      }

      thead {
        display: table-header-group;
      }

      tr {
        break-inside: avoid;
        page-break-inside: avoid;
      }

      th, td {
        border: 1px solid var(--line);
        padding: 2.2mm 2.6mm;
        text-align: left;
        vertical-align: top;
        overflow-wrap: anywhere;
      }

      th {
        background: var(--forest-soft);
        color: var(--forest-deep);
        font-weight: 700;
      }

      tbody tr:nth-child(even) td {
        background: #f8faf8;
      }

      pre {
        margin: 3mm 0 5mm;
        padding: 3.5mm 4mm;
        border: 1px solid #294b3c;
        border-radius: 2mm;
        background: #14251e;
        color: #edf5f0;
        font-family: Consolas, "Courier New", monospace;
        font-size: 8.3pt;
        line-height: 1.5;
        white-space: pre-wrap;
        overflow-wrap: anywhere;
        break-inside: avoid;
        page-break-inside: avoid;
      }

      code {
        padding: 0.25mm 1mm;
        border-radius: 1mm;
        background: #e9efeb;
        color: #214b39;
        font-family: Consolas, "Courier New", monospace;
        font-size: 0.91em;
      }

      pre code {
        padding: 0;
        background: transparent;
        color: inherit;
        font-size: inherit;
      }

      hr {
        margin: 9mm 0;
        border: 0;
        border-top: 1px solid var(--line);
      }

      @media print {
        a {
          color: var(--forest);
        }
      }
    </style>
  </head>
  <body>
    <section class="cover">
      <div>
        <div class="cover-kicker">Deployment runbook</div>
        <div class="cover-rule"></div>
        <h1>${escapeHtml(title)}</h1>
        <h2>${escapeHtml(subtitle)}</h2>
        <div class="cover-gates">
          <span>G0 Baseline</span>
          <span>G1 Approval</span>
          <span>G2 Local gates</span>
          <span>G3 Netlify</span>
          <span>G4 Preview QA</span>
          <span>G5 Go/No-Go</span>
          <span>G6 Production</span>
          <span>G7 Operations</span>
          <span>G8 Closure</span>
        </div>
      </div>
      <div class="cover-meta">
        <div><small>Versi</small><strong>${escapeHtml(documentVersion)}</strong></div>
        <div><small>Tanggal</small><strong>${escapeHtml(documentDate)}</strong></div>
        <div><small>Target</small><strong>Netlify Free</strong></div>
        <div><small>Scope</small><strong>Phase 1A</strong></div>
      </div>
    </section>
    <main>${body}</main>
  </body>
</html>`;
}

async function main() {
  if (!existsSync(markdownPath)) {
    throw new Error(`File Markdown tidak ditemukan: ${markdownPath}`);
  }

  const markdown = readFileSync(markdownPath, "utf8");
  const html = buildDocument(markdown);
  const baseTemp = await realpath(tmpdir());
  const tempDirectory = await mkdtemp(join(baseTemp, "tauco-plan-pdf-"));
  const resolvedTempDirectory = resolve(tempDirectory);

  if (
    resolvedTempDirectory !== baseTemp &&
    !resolvedTempDirectory.startsWith(`${baseTemp}\\`) &&
    !resolvedTempDirectory.startsWith(`${baseTemp}/`)
  ) {
    throw new Error("Temporary PDF directory berada di luar OS temp.");
  }

  const htmlPath = join(tempDirectory, "plan.html");
  writeFileSync(htmlPath, html, "utf8");

  try {
    if (existsSync(pdfPath)) {
      try {
        rmSync(pdfPath);
      } catch (error) {
        if (
          error instanceof Error &&
          "code" in error &&
          ["EBUSY", "EPERM"].includes(error.code)
        ) {
          throw new Error(
            "plan.pdf sedang dibuka aplikasi lain. Tutup PDF tersebut, lalu jalankan kembali npm.cmd run plan:pdf.",
          );
        }

        throw error;
      }
    }

    const browser = await chromium.launch({ headless: true });

    try {
      const page = await browser.newPage();
      await page.goto(pathToFileURL(htmlPath).href, {
        waitUntil: "load",
      });
      await page.emulateMedia({ media: "print" });
      await page.pdf({
        path: pdfPath,
        format: "A4",
        printBackground: true,
        preferCSSPageSize: true,
      });
    } finally {
      await browser.close();
    }

    if (!existsSync(pdfPath) || statSync(pdfPath).size < 10_000) {
      throw new Error("PDF tidak terbentuk atau ukurannya tidak valid.");
    }

    console.log(
      `Berhasil membuat ${basename(pdfPath)} (${statSync(pdfPath).size.toLocaleString("id-ID")} byte).`,
    );
  } finally {
    rmSync(resolvedTempDirectory, { recursive: true, force: true });
  }
}

const entryPath = process.argv[1] ? resolve(process.argv[1]) : "";
const currentPath = fileURLToPath(import.meta.url);

if (entryPath.toLowerCase() === currentPath.toLowerCase()) {
  main().catch((error) => {
    console.error(error instanceof Error ? error.message : error);
    process.exitCode = 1;
  });
}
