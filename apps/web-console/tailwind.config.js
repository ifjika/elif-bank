/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          primary: '#0F172A', // Slate 900
          accent: '#2563EB',  // Blue 600
          surface: '#F8FAFC', // Slate 50
        },
        status: {
          success: '#16A34A', // Green 600
          warning: '#D97706', // Amber 600
          danger: '#DC2626',  // Red 600
        }
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Menlo', 'monospace'],
      }
    },
  },
  plugins: [],
}
