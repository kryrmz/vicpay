import { ScanLine } from 'lucide-react'
import { QRCodeSVG } from 'qrcode.react'
import { useState } from 'react'
import { BottomSheet } from '../../components/ui/BottomSheet'
import { Button } from '../../components/ui/Button'
import { Card } from '../../components/ui/Card'
import { useAuthStore } from '../../store/auth.store'

export function QrPage() {
  const user = useAuthStore((state) => state.user)
  const [scanSheetOpen, setScanSheetOpen] = useState(false)
  const qrValue = `vicpay://pay/${user?.id ?? 'demo'}`

  return (
    <div className="flex flex-col items-center gap-6 px-4 py-8">
      <header className="text-center">
        <h1 className="text-xl font-bold text-fg">Tu codigo QR</h1>
        <p className="mt-1 text-sm text-fg-secondary">Compartelo para recibir pagos al instante.</p>
      </header>

      <Card className="flex flex-col items-center gap-4">
        <div className="rounded-xl bg-white p-4">
          <QRCodeSVG value={qrValue} size={200} />
        </div>
        <p className="text-sm font-medium text-fg-secondary">{user?.phoneMasked}</p>
      </Card>

      <Button variant="secondary" className="w-full max-w-xs" onClick={() => setScanSheetOpen(true)}>
        <ScanLine size={18} aria-hidden="true" />
        Escanear
      </Button>

      <BottomSheet open={scanSheetOpen} onClose={() => setScanSheetOpen(false)} title="Escanear codigo">
        <p className="text-sm text-fg-secondary">
          El escaneo con camara requiere una compilacion nativa (Capacitor) y todavia no esta disponible en esta
          version web. Estamos trabajando en ello.
        </p>
      </BottomSheet>
    </div>
  )
}
