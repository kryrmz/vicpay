export type ActivityFilter = 'all' | 'in' | 'out'

interface FilterOption {
  value: ActivityFilter
  label: string
}

const OPTIONS: FilterOption[] = [
  { value: 'all', label: 'Todos' },
  { value: 'in', label: 'Ingresos' },
  { value: 'out', label: 'Egresos' },
]

export interface ActivityFiltersProps {
  value: ActivityFilter
  onChange: (filter: ActivityFilter) => void
}

export function ActivityFilters({ value, onChange }: ActivityFiltersProps) {
  return (
    <div className="flex gap-2" role="tablist" aria-label="Filtrar movimientos">
      {OPTIONS.map((option) => (
        <button
          key={option.value}
          type="button"
          role="tab"
          aria-selected={value === option.value}
          onClick={() => onChange(option.value)}
          className={`rounded-full px-3.5 py-1.5 text-sm font-medium transition-colors ${
            value === option.value ? 'bg-brand-500 text-white' : 'bg-surface-sunken text-fg-secondary'
          }`}
        >
          {option.label}
        </button>
      ))}
    </div>
  )
}
