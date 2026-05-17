# Opus Visual Brand Guidelines

This document establishes the official visual design standards for the Opus platform. All user interface components, views, layout alignments, and styling implementations within the `dash/` application must strictly adhere to these specifications to ensure a premium, cohesive, and highly polished user experience.

---

## 1. Visual Vibe & Identity

*   **Design Philosophy**: "Charcoal & Rust" — A warm, earthy, and humanist minimalist approach.
*   **Aesthetic Principle**: To bridge the gap between technical, high-tech agent systems and human warmth. The interface utilizes a carefully curated warm-neutral palette that resembles reading a beautifully printed book in a peaceful, natural setting, drastically reducing visual fatigue while maintaining a crisp, professional, and state-of-the-art SaaS feel.
*   **Core Pillars**:
    *   *High Contrast, Low Fatigue*: Stark contrast between text and backgrounds, utilizing off-white and deep warm-stone charcoal instead of blinding white and harsh black.
    *   *Tactile Warmth*: Subtle earth-tone highlights that draw focus naturally and feel physically grounded.
    *   *Sophisticated Editorial feel*: Melding beautiful geometric headings with classic editorial body typography.

---

## 2. Color System

The color palette consists of foundational background/surface colors, highly readable muted values, and strategic warm accent highlights.

| Color Token | Function / Usage | Hex Code | HSL Value |
| :--- | :--- | :--- | :--- |
| **`brand-dark`** | Default text on light mode; Primary background on dark mode | `#1c1917` | `hsl(24, 10%, 10%)` |
| **`brand-light`** | Default background on light mode; Primary text on dark mode | `#f5f4f0` | `hsl(48, 17%, 95%)` |
| **`brand-primary`** | Main call-to-action (CTA), active navigation, focus indicators | `#ea580c` | `hsl(20, 83%, 48%)` |
| **`brand-secondary`**| Interactive hover states, warning status indicators | `#d97706` | `hsl(35, 92%, 43%)` |
| **`brand-muted`** | Secondary description text, borders, placeholders | `#878580` | `hsl(45, 3%, 51%)` |
| **`brand-subtle`** | Panel backgrounds, sidebar hover surfaces, separator borders | `#e8e6e0` | `hsl(45, 9%, 90%)` |
| **`brand-success`** | System online status, completed jobs, connected integrations | `#84cc16` | `hsl(84, 81%, 44%)` |

---

## 3. Typography & Hierarchy

We employ a distinct font pairing that emphasizes readability, clean modern geometric layout structure, and elegant textual content presentation.

### 3.1 Font Stack
1.  **Headings (Sans-Serif)**: `Poppins`
    *   *Fallbacks*: `system-ui, -apple-system, sans-serif`
    *   *Role*: Used for all headers, titles, metrics, and navigation elements. Bold and geometric.
2.  **Body Text (Serif)**: `Lora`
    *   *Fallbacks*: `Georgia, Cambria, serif`
    *   *Role*: Used for long-form descriptions, documentation, server log explanations, and detailed tables. Focuses on readable elegance.
3.  **Technical Logs (Monospace)**: `JetBrains Mono`
    *   *Fallbacks*: `Fira Code, SFMono-Regular, Consolas, monospace`
    *   *Role*: Used for terminal logs, code blocks, raw data outputs, and configuration inputs.

### 3.2 Type Scale

```
h1 (Page Title)      -> 2.000rem (32px) | Bold (font-bold)       | font-sans
h2 (Section Title)   -> 1.500rem (24px) | Semibold (font-semibold)| font-sans
h3 (Sub-section)     -> 1.250rem (20px) | Medium (font-medium)   | font-sans
body-base (Paragraph)-> 1.000rem (16px) | Normal (font-normal)   | font-serif | leading-relaxed (1.625)
body-sm (Muted info) -> 0.875rem (14px) | Normal (font-normal)   | font-serif
code-base (Technical)-> 0.875rem (14px) | Regular                | font-mono
```

---

## 4. Corner Roundness (Border Radius)

To maintain a balanced, structured, yet organic interface, we use a balanced, subtle curvature scale:

*   **Interactive Controls (Buttons, Text Inputs, Selects)**: `rounded-md` (equivalent to `0.375rem` or `6px`). Provides a firm, structural, and sharp professional aesthetic when aligned in lists or toolbars.
*   **Content Containers (Cards, Modals, Panels, Sidebars)**: `rounded-lg` (equivalent to `0.5rem` or `8px`). Creates a distinct, softer boundary grouping for UI layouts.
*   **Badges and Small Tags**: `rounded-sm` (equivalent to `0.25rem` or `4px`). Sharp and clean.

---

## 5. Tailwind CSS v4 & CSS Theme Configuration

In Tailwind CSS v4, theme customization is defined entirely in the primary CSS file using the `@theme` directive, bypassing the legacy JS config files.

Add the following to `dash/src/index.css`:

```css
@import "tailwindcss";

@theme {
  /* Typography Config */
  --font-sans: "Poppins", system-ui, -apple-system, sans-serif;
  --font-serif: "Lora", Georgia, Cambria, serif;
  --font-mono: "JetBrains Mono", monospace;

  /* Custom Color Tokens */
  --color-brand-dark: #1c1917;
  --color-brand-light: #f5f4f0;
  --color-brand-primary: #ea580c;
  --color-brand-secondary: #d97706;
  --color-brand-muted: #878580;
  --color-brand-subtle: #e8e6e0;
  --color-brand-success: #84cc16;

  /* Border Radius Tokens */
  --radius-btn: 0.375rem;    /* rounded-btn */
  --radius-card: 0.5rem;     /* rounded-card */
  --radius-badge: 0.25rem;   /* rounded-badge */
}
```

---

## 6. UI Component & Interaction Patterns

Consistency in user interaction is key to a polished visual system. All components must share uniform states and transitions.

### 6.1 Button Specifications
Buttons must use smooth transitions (`transition-all duration-200`) and clear interactive feedbacks.

*   **Primary Button**:
    *   *Classes*: `bg-brand-primary text-brand-light rounded-btn px-4 py-2 font-sans font-medium transition-all duration-200 hover:bg-brand-secondary active:scale-98 focus-visible:ring-2 focus-visible:ring-brand-primary/50`
    *   *Behavior*: Used for primary form submissions, initiating actions, or adding new items.
*   **Secondary Button**:
    *   *Classes*: `bg-brand-light text-brand-dark border border-brand-subtle rounded-btn px-4 py-2 font-sans transition-all duration-200 hover:bg-brand-subtle hover:text-brand-dark`
    *   *Behavior*: Used for cancellations, secondary options, or auxiliary panels.
*   **Ghost Button**:
    *   *Classes*: `text-brand-muted rounded-btn px-3 py-1.5 font-sans transition-all duration-200 hover:bg-brand-subtle/50 hover:text-brand-dark`
    *   *Behavior*: Used for top navigation tabs or subtle secondary buttons that should not draw background weight.

### 6.2 Text Inputs & Forms
*   **Standard Input**:
    *   *Classes*: `bg-brand-light text-brand-dark border border-brand-subtle rounded-btn px-3 py-2 font-serif placeholder:text-brand-muted transition-all focus:outline-none focus:border-brand-primary focus:ring-2 focus:ring-brand-primary/20`
    *   *Behavior*: Form controls use the `font-serif` stack for textual input, providing a soft editorial feeling to typing.

### 6.3 Card & Panel Layouts
*   **Standard Content Card**:
    *   *Classes*: `bg-brand-light text-brand-dark border border-brand-subtle rounded-card shadow-sm p-6`
*   **Interactive/Hoverable Card**:
    *   *Classes*: `bg-brand-light text-brand-dark border border-brand-subtle rounded-card shadow-sm p-6 transition-all duration-200 hover:-translate-y-0.5 hover:shadow-md hover:border-brand-muted`

### 6.4 Status Indicators
*   **Active/Online Badge**:
    *   *Classes*: `inline-flex items-center text-xs font-sans font-medium bg-brand-success/10 text-brand-success border border-brand-success/20 rounded-badge px-2 py-0.5`
*   **Offline/Alert Badge**:
    *   *Classes*: `inline-flex items-center text-xs font-sans font-medium bg-brand-primary/10 text-brand-primary border border-brand-primary/20 rounded-badge px-2 py-0.5`
