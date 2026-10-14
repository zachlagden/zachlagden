import { createHash } from "node:crypto";
import { readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const skillsDirectory = dirname(fileURLToPath(import.meta.url));
const readmePath = join(skillsDirectory, "..", "README.md");
const { groups } = JSON.parse(readFileSync(join(skillsDirectory, "skills.json"), "utf8"));
const icons = JSON.parse(readFileSync(join(skillsDirectory, "icons.json"), "utf8"));

const baseLayout = {
  fontSize: 13,
  labelSize: 11,
  labelLineHeight: 14,
  labelLetterSpacing: 0.8,
  pillHeight: 32,
  pillGap: 8,
  pillRadius: 8,
  iconSize: 16,
  paddingLeft: 11,
  paddingRight: 12,
  iconGap: 7,
  labelToPills: 10,
};

const variants = {
  wide: { suffix: "", width: 840, labelsBeside: true, labelColumn: 160, labelWrapWidth: 150, groupGap: 18, evenRows: true },
  narrow: { suffix: "-narrow", width: 480, labelsBeside: false, groupGap: 24, evenRows: false },
};

const narrowScreenQuery = "(max-width: 600px)";

const themes = {
  light: { ink: "#1f2328", muted: "#59636e", pillFill: "#f6f8fa", pillStroke: "#d1d9e0" },
  dark: { ink: "#f0f6fc", muted: "#9198a1", pillFill: "#151b23", pillStroke: "#3d444d" },
};

const fontFamily = '-apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans", Helvetica, Arial, sans-serif';
const minimumIconContrast = 3.5;

function relativeLuminance(hex) {
  const channels = [0, 2, 4].map((offset) => {
    const value = parseInt(hex.slice(offset, offset + 2), 16) / 255;
    return value <= 0.03928 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
}

function contrastRatio(firstHex, secondHex) {
  const [lighter, darker] = [relativeLuminance(firstHex), relativeLuminance(secondHex)].sort((a, b) => b - a);
  return (lighter + 0.05) / (darker + 0.05);
}

function estimateTextWidth(text) {
  let width = 0;
  for (const character of text) {
    if ("ijlt.fr' ".includes(character)) width += 0.32;
    else if ("mwMW".includes(character)) width += 0.88;
    else if (character >= "A" && character <= "Z") width += 0.68;
    else width += 0.57;
  }
  return Math.ceil(width * baseLayout.fontSize * 1.04);
}

function estimateLabelWidth(text) {
  return Math.ceil(text.length * (baseLayout.labelSize * 0.66 + baseLayout.labelLetterSpacing));
}

function escapeXml(text) {
  return text.replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");
}

function iconColour(icon, theme) {
  const pillFill = theme.pillFill.slice(1);
  if (icon.hex && contrastRatio(icon.hex, pillFill) >= minimumIconContrast) return `#${icon.hex}`;
  return theme.ink;
}

function renderIcon(item, x, y, theme) {
  const icon = icons[item.icon];
  if (!icon) throw new Error(`No icon named "${item.icon}" for ${item.name}`);
  const fill = icon.original ? "" : ` fill="${iconColour(icon, theme)}"`;
  return `<svg x="${x}" y="${y}" width="${baseLayout.iconSize}" height="${baseLayout.iconSize}" viewBox="${icon.viewBox}"${fill}>${icon.svg}</svg>`;
}

function renderPill(item, x, y, width, theme) {
  const iconX = x + baseLayout.paddingLeft;
  const iconY = y + (baseLayout.pillHeight - baseLayout.iconSize) / 2;
  const textSlotStart = iconX + baseLayout.iconSize + baseLayout.iconGap;
  const textSlotWidth = width - (textSlotStart - x) - baseLayout.paddingRight;
  return [
    `<rect x="${x + 0.5}" y="${y + 0.5}" width="${width - 1}" height="${baseLayout.pillHeight - 1}" rx="${baseLayout.pillRadius}" fill="${theme.pillFill}" stroke="${theme.pillStroke}"/>`,
    renderIcon(item, iconX, iconY, theme),
    `<text x="${textSlotStart + textSlotWidth / 2}" y="${y + baseLayout.pillHeight / 2 + 4.5}" text-anchor="middle" font-size="${baseLayout.fontSize}" fill="${theme.ink}">${escapeXml(item.name)}</text>`,
  ].join("");
}

function pillWidth(item) {
  return baseLayout.paddingLeft + baseLayout.iconSize + baseLayout.iconGap + estimateTextWidth(item.name) + baseLayout.paddingRight;
}

function rowWidth(pills) {
  return pills.reduce((total, pill) => total + pill.width, 0) + (pills.length - 1) * baseLayout.pillGap;
}

function greedyRows(pills, availableWidth) {
  const rows = [[]];
  for (const pill of pills) {
    const currentRow = rows[rows.length - 1];
    if (currentRow.length > 0 && rowWidth([...currentRow, pill]) > availableWidth) rows.push([pill]);
    else currentRow.push(pill);
  }
  return rows;
}

function evenestRows(pills, availableWidth, rowCount) {
  let best = null;
  const search = (start, rowsLeft, rows) => {
    if (rowsLeft === 1) {
      const candidate = [...rows, pills.slice(start)];
      const widths = candidate.map(rowWidth);
      if (Math.max(...widths) > availableWidth) return;
      const spread = Math.max(...widths) - Math.min(...widths);
      if (best === null || spread < best.spread) best = { rows: candidate, spread };
      return;
    }
    for (let end = start + 1; end <= pills.length - (rowsLeft - 1); end++) {
      search(end, rowsLeft - 1, [...rows, pills.slice(start, end)]);
    }
  };
  search(0, rowCount, []);
  return best ? best.rows : greedyRows(pills, availableWidth);
}

function splitIntoRows(items, availableWidth, evenRows) {
  const pills = items.map((item) => ({ item, width: pillWidth(item) }));
  const greedy = greedyRows(pills, availableWidth);
  return evenRows && greedy.length > 1 ? evenestRows(pills, availableWidth, greedy.length) : greedy;
}

function stretchToWidth(row, availableWidth) {
  const extra = availableWidth - rowWidth(row);
  const share = Math.floor(extra / row.length);
  const remainder = extra - share * row.length;
  return row.map((pill, index) => ({ ...pill, width: pill.width + share + (index < remainder ? 1 : 0) }));
}

function padEvenly(rows, availableWidth) {
  const room = rows.map((row) => availableWidth - rowWidth(row));
  const limitingIndex = room.indexOf(Math.min(...room));
  const limitingRow = rows[limitingIndex];
  const share = Math.floor(room[limitingIndex] / limitingRow.length);
  const remainder = room[limitingIndex] - share * limitingRow.length;
  return rows.map((row, rowIndex) =>
    row.map((pill, index) => ({
      ...pill,
      width: pill.width + share + (rowIndex === limitingIndex && index < remainder ? 1 : 0),
    })),
  );
}

function wrapLabel(label, maxWidth) {
  const lines = [];
  for (const word of label.split(" ")) {
    const last = lines.length - 1;
    if (last >= 0 && estimateLabelWidth(`${lines[last]} ${word}`) <= maxWidth) lines[last] = `${lines[last]} ${word}`;
    else lines.push(word);
  }
  return lines;
}

function renderLabel(lines, x, firstBaseline, theme) {
  return lines
    .map((line, index) => `<text x="${x}" y="${firstBaseline + index * baseLayout.labelLineHeight}" font-size="${baseLayout.labelSize}" font-weight="600" letter-spacing="${baseLayout.labelLetterSpacing}" fill="${theme.muted}">${escapeXml(line)}</text>`)
    .join("");
}

function renderGroup(group, top, theme, variant) {
  const label = group.label.toUpperCase();
  const pillsLeft = variant.labelsBeside ? variant.labelColumn : 0;
  const availableWidth = variant.width - pillsLeft;
  const splitRows = splitIntoRows(group.items, availableWidth, variant.evenRows);
  const rows = variant.evenRows
    ? padEvenly(splitRows, availableWidth)
    : splitRows.map((row, index) => (index === splitRows.length - 1 ? row : stretchToWidth(row, availableWidth)));
  const parts = [];
  let y = top;
  if (variant.labelsBeside) {
    const lines = wrapLabel(label, variant.labelWrapWidth);
    const firstBaseline = top + baseLayout.pillHeight / 2 + 4 - ((lines.length - 1) * baseLayout.labelLineHeight) / 2;
    parts.push(renderLabel(lines, 0, firstBaseline, theme));
  } else {
    parts.push(renderLabel([label], 0, top + baseLayout.labelSize, theme));
    y = top + baseLayout.labelSize + baseLayout.labelToPills;
  }
  rows.forEach((row, rowIndex) => {
    let x = pillsLeft;
    for (const pill of row) {
      parts.push(renderPill(pill.item, x, y, pill.width, theme));
      x += pill.width + baseLayout.pillGap;
    }
    y += baseLayout.pillHeight + (rowIndex === rows.length - 1 ? 0 : baseLayout.pillGap);
  });
  return { markup: parts.join(""), bottom: y };
}

function describeItem(item) {
  return item.detail ? `${item.name} (${item.detail})` : item.name;
}

function describeGroups() {
  return groups.map((group) => `${group.label}: ${group.items.map(describeItem).join(", ")}.`).join(" ");
}

function renderSvg(theme, variant) {
  const body = [];
  let top = 0;
  for (const group of groups) {
    const { markup, bottom } = renderGroup(group, top, theme, variant);
    body.push(markup);
    top = bottom + variant.groupGap;
  }
  const height = top - variant.groupGap;
  const description = escapeXml(describeGroups());
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${variant.width}" height="${height}" viewBox="0 0 ${variant.width} ${height}" font-family='${fontFamily}' role="img" aria-labelledby="title description"><title id="title">What I work with</title><desc id="description">${description}</desc>${body.join("")}</svg>\n`;
}

function renderPictureMarkup(files) {
  const alt = escapeXml(`What I work with. ${describeGroups()}`);
  const path = (variantName, themeName) => `skills/${files[variantName][themeName]}`;
  return [
    "<picture>",
    `  <source media="(prefers-color-scheme: dark) and ${narrowScreenQuery}" srcset="${path("narrow", "dark")}">`,
    `  <source media="${narrowScreenQuery}" srcset="${path("narrow", "light")}">`,
    `  <source media="(prefers-color-scheme: dark)" srcset="${path("wide", "dark")}">`,
    `  <img src="${path("wide", "light")}" alt="${alt}">`,
    "</picture>",
  ].join("\n");
}

function updateReadme(files) {
  const readme = readFileSync(readmePath, "utf8");
  const pattern = /<!-- skills:start -->[\s\S]*?<!-- skills:end -->/;
  if (!pattern.test(readme)) {
    console.warn("README.md has no skills markers, so it was left unchanged.");
    return;
  }
  const replacement = `<!-- skills:start -->\n${renderPictureMarkup(files)}\n<!-- skills:end -->`;
  writeFileSync(readmePath, readme.replace(pattern, () => replacement));
}

function contentHash(content) {
  return createHash("sha256").update(content).digest("hex").slice(0, 8);
}

function removePreviousGraphics() {
  const graphicName = /^skills(-narrow)?-(light|dark)(\.[0-9a-f]{8})?\.svg$/;
  for (const file of readdirSync(skillsDirectory)) {
    if (graphicName.test(file)) rmSync(join(skillsDirectory, file));
  }
}

removePreviousGraphics();
const files = {};
for (const [variantName, variant] of Object.entries(variants)) {
  files[variantName] = {};
  for (const [themeName, theme] of Object.entries(themes)) {
    const svg = renderSvg(theme, variant);
    const file = `skills${variant.suffix}-${themeName}.${contentHash(svg)}.svg`;
    writeFileSync(join(skillsDirectory, file), svg);
    files[variantName][themeName] = file;
  }
}
updateReadme(files);
console.log(`Wrote ${Object.values(files).flatMap((themeFiles) => Object.values(themeFiles)).join(", ")}.`);
