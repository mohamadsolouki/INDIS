import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import fa from './locales/fa.json'
import en from './locales/en.json'
import fr from './locales/fr.json'

i18n.use(initReactI18next).init({
  resources: {
    fa: { translation: fa },
    en: { translation: en },
    fr: { translation: fr },
  },
  lng: 'fa',
  fallbackLng: 'en',
  interpolation: { escapeValue: false },
})

export default i18n
