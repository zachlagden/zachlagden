import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const skillsDirectory = dirname(fileURLToPath(import.meta.url));
const readmePath = join(skillsDirectory, "..", "README.md");
const { groups } = JSON.parse(readFileSync(join(skillsDirectory, "skills.json"), "utf8"));
const icons = JSON.parse(readFileSync(join(skillsDirectory, "icons.json"), "utf8"));

const layout = {
  width: 480,
  fontSize: 13,
  labelSize: 11,
  pillHeight: 32,
  pillGap: 8,
  pillRadius: 8,
  iconSize: 16,
  paddingLeft: 11,
  paddingRight: 12,
  iconGap: 7,
  groupGap: 24,
  labelToPills: 10,
};

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
  return Math.ceil(width * layout.fontSize * 1.04);
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
  return `<svg x="${x}" y="${y}" width="${layout.iconSize}" height="${layout.iconSize}" viewBox="${icon.viewBox}"${fill}>${icon.svg}</svg>`;
}

function renderPill(item, x, y, width, theme) {
  const iconX = x + layout.paddingLeft;
  const iconY = y + (layout.pillHeight - layout.iconSize) / 2;
  const textSlotStart = iconX + layout.iconSize + layout.iconGap;
  const textSlotWidth = width - (textSlotStart - x) - layout.paddingRight;
  return [
    `<rect x="${x + 0.5}" y="${y + 0.5}" width="${width - 1}" height="${layout.pillHeight - 1}" rx="${layout.pillRadius}" fill="${theme.pillFill}" stroke="${theme.pillStroke}"/>`,
    renderIcon(item, iconX, iconY, theme),
    `<text x="${textSlotStart + textSlotWidth / 2}" y="${y + layout.pillHeight / 2 + 4.5}" text-anchor="middle" font-size="${layout.fontSize}" fill="${theme.ink}">${escapeXml(item.name)}</text>`,
  ].join("");
}

function pillWidth(item) {
  return layout.paddingLeft + layout.iconSize + layout.iconGap + estimateTextWidth(item.name) + layout.paddingRight;
}

function splitIntoRows(items) {
  const rows = [[]];
  let used = 0;
  for (const item of items) {
    const width = pillWidth(item);
    const currentRow = rows[rows.length - 1];
    const needed = currentRow.length === 0 ? width : used + layout.pillGap + width;
    if (currentRow.length > 0 && needed > layout.width) {
      rows.push([{ item, width }]);
      used = width;
    } else {
      currentRow.push({ item, width });
      used = needed;
    }
  }
  return rows;
}

function stretchToFullWidth(row) {
  const gaps = (row.length - 1) * layout.pillGap;
  const natural = row.reduce((total, pill) => total + pill.width, 0) + gaps;
  const extra = layout.width - natural;
  const share = Math.floor(extra / row.length);
  const remainder = extra - share * row.length;
  return row.map((pill, index) => ({ ...pill, width: pill.width + share + (index < remainder ? 1 : 0) }));
}

function renderGroup(group, top, theme) {
  const parts = [
    `<text x="0" y="${top + layout.labelSize}" font-size="${layout.labelSize}" font-weight="600" letter-spacing="0.8" fill="${theme.muted}">${escapeXml(group.label.toUpperCase())}</text>`,
  ];
  const rows = splitIntoRows(group.items);
  let y = top + layout.labelSize + layout.labelToPills;
  rows.forEach((row, rowIndex) => {
    const isLastRow = rowIndex === rows.length - 1;
    const pills = isLastRow ? row : stretchToFullWidth(row);
    let x = 0;
    for (const pill of pills) {
      parts.push(renderPill(pill.item, x, y, pill.width, theme));
      x += pill.width + layout.pillGap;
    }
    y += layout.pillHeight + (isLastRow ? 0 : layout.pillGap);
  });
  return { markup: parts.join(""), bottom: y };
}

function describeItem(item) {
  return item.detail ? `${item.name} (${item.detail})` : item.name;
}

function describeGroups() {
  return groups.map((group) => `${group.label}: ${group.items.map(describeItem).join(", ")}.`).join(" ");
}

function renderSvg(theme) {
  const body = [];
  let top = 0;
  for (const group of groups) {
    const { markup, bottom } = renderGroup(group, top, theme);
    body.push(markup);
    top = bottom + layout.groupGap;
  }
  const height = top - layout.groupGap;
  const description = escapeXml(describeGroups());
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${layout.width}" height="${height}" viewBox="0 0 ${layout.width} ${height}" font-family='${fontFamily}' role="img" aria-labelledby="title description"><title id="title">What I work with</title><desc id="description">${description}</desc>${body.join("")}</svg>\n`;
}

function renderPictureMarkup() {
  const alt = escapeXml(`What I work with. ${describeGroups()}`);
  return [
    "<picture>",
    '  <source media="(prefers-color-scheme: dark)" srcset="skills/skills-dark.svg">',
    `  <img src="skills/skills-light.svg" alt="${alt}" width="${layout.width}">`,
    "</picture>",
  ].join("\n");
}

function updateReadme() {
  const readme = readFileSync(readmePath, "utf8");
  const pattern = /<!-- skills:start -->[\s\S]*?<!-- skills:end -->/;
  if (!pattern.test(readme)) {
    console.warn("README.md has no skills markers, so it was left unchanged.");
    return;
  }
  const replacement = `<!-- skills:start -->\n${renderPictureMarkup()}\n<!-- skills:end -->`;
  writeFileSync(readmePath, readme.replace(pattern, () => replacement));
}

for (const [name, theme] of Object.entries(themes)) {
  writeFileSync(join(skillsDirectory, `skills-${name}.svg`), renderSvg(theme));
}
updateReadme();
console.log("Wrote skills-light.svg and skills-dark.svg.");
