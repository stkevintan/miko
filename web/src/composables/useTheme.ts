import { ref, onMounted } from 'vue';

export function useTheme() {
  const isDark = ref(false);

  const updateTheme = (dark: boolean) => {
    isDark.value = dark;
    if (dark) {
      document.documentElement.classList.add('dark');
      localStorage.setItem('theme', 'dark');
    } else {
      document.documentElement.classList.remove('dark');
      localStorage.setItem('theme', 'light');
    }
  };

  const toggleDarkMode = () => {
    updateTheme(!isDark.value);
  };

  const initTheme = () => {
    const savedTheme = localStorage.getItem('theme');
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    const shouldBeDark = savedTheme === 'dark' || (!savedTheme && prefersDark);
    
    isDark.value = shouldBeDark;
    if (shouldBeDark) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  };

  onMounted(() => {
    initTheme();
  });

  return {
    isDark,
    toggleDarkMode,
    initTheme
  };
}
