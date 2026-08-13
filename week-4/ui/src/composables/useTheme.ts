import { ref } from 'vue'

const THEME_KEY = 'paas_theme'
type Theme = 'dark' | 'light'

const media = window.matchMedia('(prefers-color-scheme: dark)')
const stored = localStorage.getItem(THEME_KEY) as Theme | null

const theme = ref<Theme>(stored ?? (media.matches ? 'dark' : 'light'))

// An explicit stored choice stamps data-theme, which always wins over the
// OS preference (see main.css). No stored choice means "system" — leave the
// attribute unset and let the prefers-color-scheme media query decide.
if (stored) {
  document.documentElement.setAttribute('data-theme', stored)
}

media.addEventListener('change', (e) => {
  if (!localStorage.getItem(THEME_KEY)) {
    theme.value = e.matches ? 'dark' : 'light'
  }
})

export function useTheme() {
  function toggle() {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
    localStorage.setItem(THEME_KEY, theme.value)
    document.documentElement.setAttribute('data-theme', theme.value)
  }

  return { theme, toggle }
}
