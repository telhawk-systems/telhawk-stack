/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        sidebar: {
          DEFAULT: '#1e2530',
          hover: '#2d3748',
          text: '#a0aec0',
          active: '#ffffff',
          accent: '#4299e1',
        },
        severity: {
          critical: {
            DEFAULT: '#c53030',
            bg: '#fff5f5',
          },
          high: {
            DEFAULT: '#dd6b20',
            bg: '#fffaf0',
          },
          medium: {
            DEFAULT: '#d69e2e',
            bg: '#fffff0',
          },
          low: {
            DEFAULT: '#38a169',
            bg: '#f0fff4',
          },
          info: {
            DEFAULT: '#3182ce',
            bg: '#ebf8ff',
          },
        },
        surface: {
          primary: '#ffffff',
          elevated: '#ffffff',
          border: '#e2e8f0',
        },
      },
      spacing: {
        'sidebar': '240px',
        'sidebar-collapsed': '64px',
        'topbar': '56px',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'Helvetica Neue', 'Arial', 'sans-serif'],
        mono: ['Roboto Mono', 'SF Mono', 'Monaco', 'Consolas', 'Liberation Mono', 'Courier New', 'monospace'],
      },
    },
  },
  plugins: [],
}
