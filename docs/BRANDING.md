# Branding & UI Guidelines

**Project:** Opus  
**Date:** 2026-05-16  
**Theme:** Earthy & Warm (Premium, approachable vibe inspired by Anthropic & Notion)

## 1. Core Colors & Typography

The application uses an "Earthy" palette to convey a premium and warm feel.

### Base Colors
- **Dark:** `#1C1917` (Used for primary text in light mode, background in dark mode)
- **Light:** `#FFFCF5` (Used for primary background in light mode, text in dark mode)
- **Mid Gray:** `#A8A29E` (Used for secondary text, disabled states, borders)
- **Light Gray:** `#F5F5F4` (Used for subtle backgrounds, secondary elements like cards)

### Accent Colors
- **Terracotta:** `#C53030` (Primary accent, calls to action, destructive actions)
- **Mustard:** `#D97706` (Warning states, highlights, secondary buttons)
- **Sage Green:** `#657B83` (Success states, positive indicators)

### Typography
- **Headings (H1-H6):** `Poppins` (Bold/Semibold) - Provides a clean, modern, and structured feel.
- **Body Text:** `Lora` (Regular/Medium) - Offers high readability and an elegant, editorial feel.
- *Fallback:* Arial for headings, Georgia for body text.

## 2. Glassmorphism Application

To maintain a clean and uncluttered interface, glassmorphism is used **sparingly and strictly** for elements that float above the main content (high Z-index).

### Authorized Elements for Glassmorphism
- **Sticky Headers / Navigation Bars** (only when scrolling)
- **Dropdown Menus & Popovers** (Command palettes, context menus)
- **Modals & Dialogs**
- **Toasts / Notifications**
- **Floating Action Buttons (FABs)**

### Technical Recipe (Tailwind CSS Guidelines)
Whenever applying a glass effect, adhere to this combination:
1. **Background Opacity:** `bg-white/70` (Light mode) or `bg-black/60` (Dark mode)
2. **Backdrop Blur:** `backdrop-blur-md` (Standardized to roughly 12px blur radius)
3. **Subtle Border:** Crucial for defining the edge of the "glass".
   - Light mode: `border border-white/40`
   - Dark mode: `border border-white/10`
4. **Shadow:** `shadow-md` to elevate the element from the content behind it.

## 3. Implementation Strategy (Next.js & Tailwind)

- **Tailwind Configuration:** Extend `tailwind.config.ts` to include the specific base and accent colors as custom properties (e.g., `theme.colors.opus.dark`, `theme.colors.opus.terracotta`).
- **Global CSS:** Define the font-family utilities in `globals.css` ensuring `Poppins` and `Lora` are imported and mapped to the respective Tailwind classes (`font-heading`, `font-body`).
- **Shadcn/ui Integration:** Override the default Shadcn/ui variables in `globals.css` to align with the Earthy theme (e.g., mapping `--primary` to Terracotta). Ensure Shadcn/ui components like Dialogs and Dropdowns are updated to include the Glassmorphism recipe defined above.