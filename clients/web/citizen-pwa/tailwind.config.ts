import type { Config } from 'tailwindcss';

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        vazir: ['Vazirmatn', 'sans-serif'],
        sans: ['Vazirmatn', 'system-ui', 'sans-serif'],
      },
      colors: {
        // Iranian national green + red palette
        iran: {
          green: '#239f40',
          'green-dark': '#006400',
          red: '#cc0001',
          'red-dark': '#8b0000',
          white: '#ffffff',
        },
        indis: {
          primary: '#1a6b3c',
          'primary-dark': '#0f4a28',
          accent: '#cc0001',
          surface: '#f8f9fa',
          border: '#e2e8f0',
        },
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-in-out',
        'slide-up': 'slideUp 0.3s ease-out',
      },
      keyframes: {
        fadeIn: { from: { opacity: '0' }, to: { opacity: '1' } },
        slideUp: { from: { transform: 'translateY(20px)', opacity: '0' }, to: { transform: 'translateY(0)', opacity: '1' } },
      },
    },
  },
  plugins: [],
} satisfies Config;
