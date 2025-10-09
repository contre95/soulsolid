/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: "class",
  content: [
    "./views/**/*.html",
    "./views/jobs/*.html",
    "./views/downloading/**/*.html",
    "./public/**/*.js",
  ],
  safelist: [
    // Add needed classes here
    "hidden",
    "invisible",
    "opacity-0",
    "backdrop-blur-md",
    "bg-opacity-20",
  ],
  theme: {
    extend: {
      colors: {
        purple: {
          50: "#f5f3ff",
          100: "#ede9fe",
          200: "#ddd6fe",
          300: "#c4b5fd",
          400: "#a78bfa",
          500: "#8b5cf6",
          600: "#7c3aed",
          700: "#6d28d9",
          800: "#5b21b6",
          900: "#4c1d95",
          950: "#2e1065",
        },
      },
    },
  },
  variants: {
    extend: {
      display: ["group-hover"],
    },
  },
  plugins: [],
};
