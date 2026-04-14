/**
 * Mermaid 11.x theme objects for Beluga AI.
 *
 * Mermaid cannot resolve CSS custom properties at init time, so the
 * values below are literal hex approximations of the OKLCH tokens in
 * `src/styles/global.css`. Keep them in sync with that file when the
 * brand palette moves.
 *
 * Verified against `mermaid@11.12.3` — the v10 keys `arrowheadColor`
 * and `signalColor` still compile (themeVariables is typed `any`),
 * but arrowheads now inherit from `lineColor` so we don't bother
 * setting them explicitly.
 *
 * Source of truth for the hex values:
 *   --brand-200 oklch(0.89 0.03 215)  → #C9E1E5
 *   --brand-500 oklch(0.68 0.09 235)  → #5CA3CA  (logo primary)
 *   --brand-700 oklch(0.52 0.10 240)  → #3B7898
 *   --brand-800 oklch(0.42 0.08 244)  → #2C5E7A
 *   --ink-950   oklch(0.12 0.012 240) → #0E1217  (dark page bg)
 *   --ink-900   oklch(0.16 0.014 240) → #141920
 *   --ink-850   oklch(0.20 0.014 238) → #1B2027
 *   --ink-800   oklch(0.26 0.012 235) → #262D35
 *   --ink-500   oklch(0.62 0.008 230) → #8B9299
 *   --ink-300   oklch(0.82 0.010 225) → #C7CBD1  (dark body text)
 *   --ink-100   oklch(0.95 0.010 220) → #EAEDF0
 *   --paper-50  oklch(0.985 0.005 215) → #F7F9FA  (light page bg)
 *   --paper-100 oklch(0.965 0.006 215) → #EEF1F3
 *   --paper-200 oklch(0.935 0.008 218) → #E1E6EA
 *   --paper-700 oklch(0.44 0.011 232) → #5C6670
 *   --paper-900 oklch(0.22 0.014 240) → #272E38
 *   --paper-950 oklch(0.14 0.016 242) → #171D27
 */

const FONT_STACK =
  '"IBM Plex Sans", system-ui, -apple-system, "Segoe UI", sans-serif';

export const belugaDark = {
  background: "#0E1217",
  mainBkg: "#141920",
  secondaryColor: "#1B2027",
  tertiaryColor: "#262D35",
  primaryColor: "#3B7898",
  primaryBorderColor: "#5CA3CA",
  primaryTextColor: "#EAEDF0",
  secondaryBorderColor: "#2C5E7A",
  secondaryTextColor: "#C7CBD1",
  tertiaryBorderColor: "#262D35",
  tertiaryTextColor: "#8B9299",

  lineColor: "#8B9299",
  edgeLabelBackground: "#141920",

  clusterBkg: "#1B2027",
  clusterBorder: "#262D35",
  titleColor: "#EAEDF0",

  nodeTextColor: "#C7CBD1",
  textColor: "#C7CBD1",

  activationBkgColor: "#3B7898",
  activationBorderColor: "#5CA3CA",

  // Sequence diagram
  actorBkg: "#141920",
  actorBorder: "#5CA3CA",
  actorTextColor: "#EAEDF0",
  actorLineColor: "#8B9299",
  signalColor: "#C7CBD1",
  signalTextColor: "#C7CBD1",
  labelBoxBkgColor: "#1B2027",
  labelBoxBorderColor: "#5CA3CA",
  labelTextColor: "#EAEDF0",
  loopTextColor: "#C7CBD1",
  noteBkgColor: "#1B2027",
  noteBorderColor: "#2C5E7A",
  noteTextColor: "#C7CBD1",

  // State diagram
  stateBkg: "#141920",
  stateBorder: "#5CA3CA",
  labelColor: "#EAEDF0",

  fontFamily: FONT_STACK,
  fontSize: "14px",
};

export const belugaLight: typeof belugaDark = {
  background: "#F7F9FA",
  mainBkg: "#EEF1F3",
  secondaryColor: "#E1E6EA",
  tertiaryColor: "#D2D9DF",
  primaryColor: "#5CA3CA",
  primaryBorderColor: "#2C5E7A",
  primaryTextColor: "#171D27",
  secondaryBorderColor: "#3B7898",
  secondaryTextColor: "#272E38",
  tertiaryBorderColor: "#E1E6EA",
  tertiaryTextColor: "#5C6670",

  lineColor: "#5C6670",
  edgeLabelBackground: "#EEF1F3",

  clusterBkg: "#EEF1F3",
  clusterBorder: "#E1E6EA",
  titleColor: "#171D27",

  nodeTextColor: "#272E38",
  textColor: "#272E38",

  activationBkgColor: "#5CA3CA",
  activationBorderColor: "#2C5E7A",

  actorBkg: "#EEF1F3",
  actorBorder: "#2C5E7A",
  actorTextColor: "#171D27",
  actorLineColor: "#5C6670",
  signalColor: "#272E38",
  signalTextColor: "#272E38",
  labelBoxBkgColor: "#E1E6EA",
  labelBoxBorderColor: "#2C5E7A",
  labelTextColor: "#171D27",
  loopTextColor: "#272E38",
  noteBkgColor: "#EEF1F3",
  noteBorderColor: "#3B7898",
  noteTextColor: "#272E38",

  stateBkg: "#EEF1F3",
  stateBorder: "#2C5E7A",
  labelColor: "#171D27",

  fontFamily: FONT_STACK,
  fontSize: "14px",
};

/** Read the theme attribute from the html element and return the matching object. */
export function getMermaidTheme(): typeof belugaDark {
  if (typeof document === "undefined") return belugaDark;
  return document.documentElement.dataset.theme === "light"
    ? belugaLight
    : belugaDark;
}
