export const memberThemes = {
  'default': {
    primary: '#6366f1',
    secondary: '#818cf8',
    gradient: 'linear-gradient(135deg, #6366f1 0%, #818cf8 100%)',
    gradientDark: 'linear-gradient(135deg, #4338ca 0%, #6366f1 100%)',
  },
  '五百城茉央': {
    primary: '#40E0D0',
    secondary: '#1E90FF',
    gradient: 'linear-gradient(135deg, #40E0D0 0%, #1E90FF 100%)',
    gradientDark: 'linear-gradient(135deg, #2aada0 0%, #1565c0 100%)',
  },
}

export function applyTheme(memberName, isDark = false) {
  const theme = memberThemes[memberName] || memberThemes['default']
  const root = document.documentElement
  root.style.setProperty('--color-primary', theme.primary)
  root.style.setProperty('--color-secondary', theme.secondary)
  root.style.setProperty('--color-gradient', isDark ? theme.gradientDark : theme.gradient)
}
