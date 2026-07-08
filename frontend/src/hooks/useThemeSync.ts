import { useEffect } from 'react'
import { useUiStore } from '../store/ui.store'

/** Refleja el tema elegido por el usuario en el atributo data-theme de <html>. */
export function useThemeSync(): void {
  const theme = useUiStore((state) => state.theme)

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
  }, [theme])
}
