import type { Config } from "tailwindcss";

export default {
  darkMode: ["class"],
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        bg: "#0D0D0F",
        card: "#141416",
        elevated: "#1C1C1F",
        border: "#2A2A2E",
        text: {
          primary: "#F0F0F0",
          muted: "#888890",
          subtle: "#555560",
        },
        provider: {
          groq: "#EF4444",
        },
        status: {
          pending: "#F59E0B",
          active: "#22C55E",
          completed: "#6B7280",
        },
      },
      borderRadius: {
        xl: "12px",
      },
      fontFamily: {
        display: ["Outfit", "sans-serif"],
        body: ["DM Sans", "sans-serif"],
        mono: ["JetBrains Mono", "monospace"],
      },
      keyframes: {
        pulseDot: {
          "0%, 100%": { transform: "scale(1)", opacity: "1" },
          "50%": { transform: "scale(1.15)", opacity: "0.6" },
        },
        shimmer: {
          "0%": { backgroundPosition: "200% 0" },
          "100%": { backgroundPosition: "-200% 0" },
        },
        slideIn: {
          "0%": { transform: "translateY(14px)", opacity: "0" },
          "100%": { transform: "translateY(0)", opacity: "1" },
        },
      },
      animation: {
        pulseDot: "pulseDot 1.5s infinite",
        shimmer: "shimmer 1.8s linear infinite",
        slideIn: "slideIn 220ms ease",
      },
    },
  },
  plugins: [],
} satisfies Config;
