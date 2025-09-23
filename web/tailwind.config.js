/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          50: '#f5f8ff',
          100: '#ebf1ff',
          200: '#ccd9ff',
          300: '#a7bbff',
          400: '#7d93ff',
          500: '#4b63ff',
          600: '#3448db',
          700: '#2736aa',
          800: '#212f85',
          900: '#1f2c6b',
        },
      },
    },
  },
  plugins: [],
}
