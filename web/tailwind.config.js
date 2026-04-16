/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: '#7C8CF8',
          light: '#A5B4FC',
          dark: '#5B6AE0',
        },
        bg: {
          deep: '#0F1117',
          surface: '#1A1D2E',
          card: '#232640',
          hover: '#2A2E4A',
        },
        text: {
          primary: '#E8EAF6',
          secondary: '#9CA3C0',
          muted: '#5C6280',
        },
        accent: {
          cyan: '#67E8F9',
          green: '#6EE7B7',
          amber: '#FCD34D',
        },
      },
      boxShadow: {
        'glow': '0 0 15px rgba(124, 140, 248, 0.1)',
        'glow-md': '0 0 25px rgba(124, 140, 248, 0.2)',
        'glow-lg': '0 0 35px rgba(124, 140, 248, 0.3)',
      },
      animation: {
        'pulse-glow': 'pulse-glow 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
      keyframes: {
        'pulse-glow': {
          '0%, 100%': { boxShadow: '0 0 15px rgba(124, 140, 248, 0.1)' },
          '50%': { boxShadow: '0 0 25px rgba(124, 140, 248, 0.3)' },
        },
      },
    },
  },
  plugins: [],
}
