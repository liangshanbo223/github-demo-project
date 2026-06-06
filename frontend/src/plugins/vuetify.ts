/**
 * plugins/vuetify.ts
 *
 * Framework documentation: https://vuetifyjs.com
 */

// Styles
import '@mdi/font/css/materialdesignicons.css'
import 'vuetify/styles/main.css'

import colors from 'vuetify/util/colors'
import { fa, en, vi, zhHans, zhHant, ru } from 'vuetify/locale'

// Composables
import { createVuetify } from 'vuetify'

// https://vuetifyjs.com/en/introduction/why-vuetify/#feature-guides
export default createVuetify({
  defaults: {
    VRow: { density: 'compact' },
    VTextField: {
      variant: 'solo-filled',
    },
    VSelect: {
      variant: 'solo-filled',
    },
    VCombobox: {
      variant: 'solo-filled',
    },
    VTextarea: {
      variant: 'solo-filled',
    },
  },
  theme: {
    defaultTheme: localStorage.getItem('theme') ?? 'system',
    themes: {
      light: {
        colors: {
          primary: '#4F46E5', // 现代靛蓝
          secondary: '#7C3AED', // 科技紫
          error: '#EF4444',
          background: '#F8FAFC', // Slate 50 浅灰蓝
          surface: '#FFFFFF',
        },
      },
      dark: {
        colors: {
          primary: '#6366F1', // 极光蓝靛
          secondary: '#8B5CF6', // 极光紫
          error: '#F87171',
          background: '#0B0F19', // 深邃暗夜蓝
          surface: '#151B2C', // 微微泛蓝的暗灰卡片色
        },
      },
    },
  },
  locale: {
    locale: localStorage.getItem("locale") ?? 'zhHans',
    fallback: 'zhHans',
    messages: { fa, en, vi, zhHans, zhHant, ru },
  },
})
